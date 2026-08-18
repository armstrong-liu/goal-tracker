#!/bin/bash
#
# Verify task: execute_scripts (shell)
#
# Puts the task to etcd, waits for the result, and verifies it against
# expected values, replicating the Go tool's rules:
#   - status must equal EXPECT_STATUS
#   - if EXPECT_RESULT is non-empty:
#       actual result is base64-decoded and trimmed
#       EXPECT_STATUS == 0 -> "contains" match
#       otherwise          -> exact match
#
# Dependencies: etcdctl, jq, base64
#
# Usage:
#   ./verify_execute_scripts.sh
#   ./verify_execute_scripts.sh -k <key>
#   ./verify_execute_scripts.sh -s 1 -r "expected output"
#   # target the local host by system uuid:
#   ./verify_execute_scripts.sh -k "$(dmidecode -s system-uuid | tr -d '[:space:]')"

set -euo pipefail

# ---- Connection defaults ----
# Endpoint IP is the local machine's IP (etcd runs on this host, listening on
# 0.0.0.0:2379); the port is fixed. etcd auth is NOT enabled: the ule3 gomang
# connects without credentials.
LOCAL_IP=$(hostname -I 2>/dev/null | awk '{print $1}')
[ -z "${LOCAL_IP}" ] && LOCAL_IP="127.0.0.1"
ENDPOINT="${LOCAL_IP}:2379"
KEY=$(dmidecode -s system-uuid 2>/dev/null | tr -d '[:space:]' || true)

# ---- Task definition ----
TASK_TYPE="execute_scripts"
EXPECT_STATUS=1
EXPECT_RESULT=""
TASK_JSON=$(cat <<'EOF'
{"type":"execute_scripts","value":{"type":"shell","content":"bm93PSQoZGF0ZSAnKyVZLSVtLSVkICVIOiVNOiVTJyk7IGVjaG8gIiRub3ciOyBkaXI9L3Jvb3QvJChkYXRlICslWSVtJWQpOyBta2RpciAtcCAiJGRpciI7IGZpbGU9IiRkaXIvJCh0ciAtZGMgJ2EtejAtOScgPC9kZXYvdXJhbmRvbSB8IGhlYWQgLWMgOCkudHh0IjsgdG91Y2ggIiRmaWxlIjsgZWNobyAiY3JlYXRlZDogJGZpbGUi"}}
EOF
)

usage() {
    cat <<EOF
Usage: $(basename "$0") [options]

Options:
  -e <endpoint>  etcd endpoint   (default: ${ENDPOINT})
  -k <key>       etcd key        (default: ${KEY})
  -s <status>    expected status (default: ${EXPECT_STATUS})
  -r <result>    expected result (default: none -> status-only check)
  -h             show help
EOF
}

while getopts "e:k:s:r:h" opt; do
    case "$opt" in
        e) ENDPOINT="$OPTARG" ;;
        k) KEY="$OPTARG" ;;
        s) EXPECT_STATUS="$OPTARG" ;;
        r) EXPECT_RESULT="$OPTARG" ;;
        h) usage; exit 0 ;;
        *) usage; exit 1 ;;
    esac
done

# KEY defaults to this host's system-uuid; -k can override it. Guard against an
# empty value (e.g. dmidecode missing / no uuid) so we never write the task
# under an empty etcd key. Placed after getopts so -k can still supply it.
[ -n "$KEY" ] || { echo "Error: cannot determine etcd key (system-uuid); pass -k <key>" >&2; exit 1; }

for dep in etcdctl jq base64; do
    command -v "$dep" >/dev/null 2>&1 || { echo "Error: ${dep} not found" >&2; exit 1; }
done

export ETCDCTL_API=3
unset ETCDCTL_USER ETCDCTL_ENDPOINTS ETCDCTL_USERNAME ETCDCTL_PASSWORD
FLAGS=(--endpoints="${ENDPOINT}")
RESULT_KEY="${KEY}_${TASK_TYPE}_result"

echo "========================================"
echo "Task:     ${TASK_TYPE}"
echo "Endpoint: ${ENDPOINT}"
echo "Key:      ${KEY}"
echo "Expect:   status=${EXPECT_STATUS} result='${EXPECT_RESULT}'"
echo "========================================"

# 1. Clear any stale result so the poll only sees this run's result
etcdctl "${FLAGS[@]}" del "${RESULT_KEY}" >/dev/null 2>&1 || true

echo "[1/3] Putting task..."
etcdctl "${FLAGS[@]}" put "${KEY}" "${TASK_JSON}" >/dev/null
echo "      written"

# 2. Poll for the result key (2s interval, 60s timeout)
echo "[2/3] Waiting for result (up to 60s)..."
RAW_VALUE=""
STATUS=""
RAW_RESULT=""
DEADLINE=$(( SECONDS + 60 ))
while (( SECONDS < DEADLINE )); do
    RAW_VALUE=$(etcdctl "${FLAGS[@]}" get "${RESULT_KEY}" --print-value-only 2>/dev/null || true)
    if [[ -n "$RAW_VALUE" ]]; then
        if [[ "$RAW_VALUE" == "1" ]]; then
            STATUS=1; RAW_RESULT=""
        elif printf '%s' "$RAW_VALUE" | jq -e . >/dev/null 2>&1; then
            STATUS=$(printf '%s' "$RAW_VALUE" | jq -r '.status')
            RAW_RESULT=$(printf '%s' "$RAW_VALUE" | jq -r '.result // ""')
        else
            STATUS=0; RAW_RESULT="$RAW_VALUE"
        fi
        break
    fi
    sleep 2
done

if [[ -z "$STATUS" ]]; then
    echo "FAIL: timed out waiting for ${RESULT_KEY}" >&2
    exit 1
fi
echo "      got result (status=${STATUS})"

# 3. Decode + verify
echo "[3/3] Verifying..."
ACTUAL=""
if [[ -n "$RAW_RESULT" ]]; then
    if DECODED=$(printf '%s' "$RAW_RESULT" | base64 -d 2>/dev/null); then
        ACTUAL=$(printf '%s' "$DECODED" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
    else
        ACTUAL="$RAW_RESULT"
    fi
fi

PASS=true
if [[ "$STATUS" != "$EXPECT_STATUS" ]]; then
    echo "  status: FAIL (expected ${EXPECT_STATUS}, got ${STATUS})"
    PASS=false
else
    echo "  status: OK (${STATUS})"
fi

if [[ -n "$EXPECT_RESULT" ]]; then
    if [[ "$EXPECT_STATUS" == "0" ]]; then
        if [[ "$ACTUAL" == *"$EXPECT_RESULT"* ]]; then
            echo "  result: OK (contains '${EXPECT_RESULT}')"
        else
            echo "  result: FAIL (actual does not contain '${EXPECT_RESULT}')"
            echo "    actual: ${ACTUAL}"
            PASS=false
        fi
    else
        if [[ "$ACTUAL" == "$EXPECT_RESULT" ]]; then
            echo "  result: OK (exact match)"
        else
            echo "  result: FAIL (actual != expected)"
            echo "    expected: ${EXPECT_RESULT}"
            echo "    actual:   ${ACTUAL}"
            PASS=false
        fi
    fi
else
    echo "  result: (not checked)"
    [[ -n "$ACTUAL" ]] && echo "    actual: ${ACTUAL}"
fi

echo "========================================"
if $PASS; then echo "RESULT: PASS"; exit 0; else echo "RESULT: FAIL"; exit 1; fi
