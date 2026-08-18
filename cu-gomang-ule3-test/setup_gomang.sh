#!/bin/bash
#
# Install and configure gomang (ule3) to talk to the local etcd.
#
# Steps:
#   1. Ensure the gomang RPM package is installed (yum install if missing)
#   2. Append an /etc/hosts entry for etcd-internal.cucloud.cn
#   3. Install the gomang service via `gomang install` (kardianos/service
#      generates /etc/systemd/system/gomang.service); skipped if already
#      installed
#   4. daemon-reload, enable and restart gomang
#   5. Wait until gomang is actually READY (functional probe)
#
# NOTE: unlike the ule4 suite, no credentials are written. The ule3 gomang has
# no username/password in its config; it connects to the etcd at
# etcd-internal.cucloud.cn:2379 (its built-in default EtcdURI), so no
# /etc/gomang/gomang.ini is required — the /etc/hosts entry below is what
# points that hostname at the local etcd.
#
# Must run as root.

set -euo pipefail

# Local IP (etcd runs on this host); derived at runtime, not hardcoded.
LOCAL_IP=$(hostname -I 2>/dev/null | awk '{print $1}')
[ -z "${LOCAL_IP}" ] && LOCAL_IP="127.0.0.1"
HOSTS_IP="${LOCAL_IP}"
HOSTS_NAME="etcd-internal.cucloud.cn"
SERVICE="gomang"
UNIT_FILE="/etc/systemd/system/gomang.service"

# ---- Root check ----
if [[ $EUID -ne 0 ]]; then
    echo "Error: must run as root (use sudo)" >&2
    exit 1
fi

# ---- 1. Ensure the gomang RPM package is installed ----
# The suite does not build gomang; it installs the delivered RPM from the
# configured yum repos. Already installed -> keep it (test the deployed
# version); missing -> install. Known env quirk: the repo may serve an
# unsigned gomang RPM ("Package gomang-...rpm is not signed") -> retry once
# with --nogpgcheck. Any other failure aborts setup (set -e), which makes
# run() report every case as skip (honest: env not ready).
echo "[1/5] Checking gomang package"
if rpm -q gomang >/dev/null 2>&1; then
    echo "      already installed: $(rpm -q gomang)"
else
    echo "      not installed, running: yum install -y gomang"
    if YUM_OUT=$(yum install -y gomang 2>&1); then
        echo "${YUM_OUT}"
    elif echo "${YUM_OUT}" | grep -q 'is not signed'; then
        echo "${YUM_OUT}"
        echo "      unsigned package; retrying once with --nogpgcheck"
        yum install -y gomang --nogpgcheck
    else
        echo "${YUM_OUT}" >&2
        echo "Error: yum install gomang failed (not a signature issue)" >&2
        exit 1
    fi
    echo "      installed: $(rpm -q gomang)"
fi

# ---- 2. Append /etc/hosts entry ----
echo "[2/5] Configuring /etc/hosts"
if grep -qF "${HOSTS_NAME}" /etc/hosts; then
    echo "      hosts entry for ${HOSTS_NAME} already present, skipping"
else
    cp -a /etc/hosts /etc/hosts.bak
    printf '\n%s\t%s\n' "${HOSTS_IP}" "${HOSTS_NAME}" >> /etc/hosts
    echo "      added: ${HOSTS_IP} ${HOSTS_NAME}"
fi

# ---- 3. Install the gomang service ----
# ule3 gomang registers itself as a systemd service through kardianos/service
# (`gomang install` writes ${UNIT_FILE}); it does not ship a static unit file.
echo "[3/5] Installing ${SERVICE} service"
if systemctl cat "${SERVICE}.service" >/dev/null 2>&1; then
    echo "      service already installed, skipping 'gomang install'"
else
    GOMANG_BIN=$(command -v gomang || true)
    if [[ -z "${GOMANG_BIN}" ]]; then
        echo "Error: gomang binary not found in PATH, cannot install service" >&2
        exit 1
    fi
    "${GOMANG_BIN}" install
    echo "      installed via '${GOMANG_BIN} install'"
fi

# ---- 4. daemon-reload + enable + restart ----
echo "[4/5] Restarting ${SERVICE}"
systemctl daemon-reload
systemctl enable "${SERVICE}" >/dev/null 2>&1 || true
systemctl restart "${SERVICE}"
systemctl status "${SERVICE}" --no-pager
echo "      restart issued"

# ---- 5. Wait until gomang is actually READY (not just systemctl 'active') ----
# systemctl reports the unit active as soon as the process forks, but gomang may
# not yet have connected to etcd / established its task watch. A case that puts a
# task during that gap can be missed -> 60s timeout -> spurious fail (hits the
# FIRST case). Probe functionally: put a no-op execute_scripts task and wait for
# gomang to write a result; re-put periodically so a task missed during the
# startup gap (watch-only model) is retried. If gomang never processes tasks,
# fail setup -> run() marks all cases 'skip' (honest: env not ready, not a bug).
echo "[5/5] Waiting for gomang to be ready (functional probe)"
export ETCDCTL_API=3
unset ETCDCTL_USER ETCDCTL_ENDPOINTS ETCDCTL_USERNAME ETCDCTL_PASSWORD
PROBE_KEY=$(dmidecode -s system-uuid 2>/dev/null | tr -d '[:space:]' || true)
if [[ -z "$PROBE_KEY" ]]; then
    echo "Error: cannot get system-uuid for gomang readiness probe" >&2
    exit 1
fi
PROBE_EP="${LOCAL_IP}:2379"
PROBE_RESULT="${PROBE_KEY}_execute_scripts_result"
PROBE_FLAGS=(--endpoints="${PROBE_EP}")
# base64("echo ready") — harmless no-op shell task (content correctness is not
# required; we only need gomang to produce SOME result to prove it is processing).
PROBE_TASK='{"type":"execute_scripts","value":{"type":"shell","content":"ZWNobyByZWFkeQ=="}}'
GOMANG_READY=false
PROBE_DEADLINE=$(( SECONDS + 180 ))
while (( SECONDS < PROBE_DEADLINE )); do
    etcdctl "${PROBE_FLAGS[@]}" del "${PROBE_RESULT}" >/dev/null 2>&1 || true
    etcdctl "${PROBE_FLAGS[@]}" put "${PROBE_KEY}" "${PROBE_TASK}" >/dev/null 2>&1 || true
    for _ in 1 2 3 4 5; do
        sleep 2
        if [[ -n "$(etcdctl "${PROBE_FLAGS[@]}" get "${PROBE_RESULT}" --print-value-only 2>/dev/null || true)" ]]; then
            GOMANG_READY=true
            break
        fi
    done
    if $GOMANG_READY; then break; fi
    echo "      not ready yet, re-putting probe task..."
done
etcdctl "${PROBE_FLAGS[@]}" del "${PROBE_RESULT}" >/dev/null 2>&1 || true
if ! $GOMANG_READY; then
    echo "Error: gomang did not process tasks within 180s (not ready)" >&2
    exit 1
fi
echo "      gomang ready"
