#!/bin/bash
#
# Install etcd and configure it to listen on all interfaces.
#
# Steps:
#   1. Install etcd via yum
#   2. Update client URLs in /etc/etcd/etcd.conf (default was localhost:2379)
#   3. Reset data dir, restart and enable etcd service
#   4. Wait until etcd is reachable
#
# NOTE: etcd authentication is NOT enabled here. The ule3 gomang connects to
# etcd WITHOUT credentials (its config has no username/password), so etcd must
# run in open mode to match.
#
# Must run as root.

set -euo pipefail

CONF="/etc/etcd/etcd.conf"
LISTEN_URL="http://0.0.0.0:2379"

# Local IP detection (etcd runs on this host)
LOCAL_IP=$(hostname -I 2>/dev/null | awk '{print $1}')
[ -z "${LOCAL_IP}" ] && LOCAL_IP="127.0.0.1"
EP="${LOCAL_IP}:2379"

# ---- Root check ----
if [[ $EUID -ne 0 ]]; then
    echo "Error: must run as root (use sudo)" >&2
    exit 1
fi

# ---- 1. Install etcd (+ jq dependency) ----
if command -v etcd >/dev/null 2>&1; then
    echo "[1/4] etcd already installed, skipping"
else
    echo "[1/4] Installing etcd..."
    yum install -y etcd
fi

# verify_*.sh needs jq to parse task results; ensure it is present.
if ! command -v jq >/dev/null 2>&1; then
    echo "      installing jq (required by verify_*.sh)..."
    yum install -y jq
fi

# ---- 2. Configure client URLs ----
if [[ ! -f "$CONF" ]]; then
    echo "Error: config file not found: $CONF" >&2
    exit 1
fi

# Backup once before first modification
if [[ ! -f "${CONF}.bak" ]]; then
    cp -a "$CONF" "${CONF}.bak"
fi

echo "[2/4] Configuring $CONF (listen + advertise URLs -> $LISTEN_URL)"

# Replace existing uncommented line, or append if not present
set_conf() {
    local key="$1" val="$2"
    if grep -q "^${key}=" "$CONF"; then
        sed -i "s|^${key}=.*|${key}=\"${val}\"|" "$CONF"
    else
        echo "${key}=\"${val}\"" >> "$CONF"
    fi
}

set_conf "ETCD_LISTEN_CLIENT_URLS" "$LISTEN_URL"
set_conf "ETCD_ADVERTISE_CLIENT_URLS" "$LISTEN_URL"

# ---- 3. Reset data dir + restart etcd ----
# Wipe the data dir BEFORE (re)starting so a previous run's persisted state
# (stale keys) does not carry over. etcd keeps ALL state under its data dir,
# which survives `yum remove` and a skipped teardown; clearing it here keeps
# every run self-contained.
DATA_DIR=$(grep -E '^ETCD_DATA_DIR=' "$CONF" 2>/dev/null | tail -1 | cut -d= -f2- | tr -d '"' || true)
[ -z "$DATA_DIR" ] && DATA_DIR="/var/lib/etcd/default.etcd"
echo "[3/4] Resetting data dir (${DATA_DIR}) and restarting etcd..."
systemctl stop etcd 2>/dev/null || true
rm -rf "${DATA_DIR}"
systemctl enable etcd >/dev/null 2>&1 || true
systemctl restart etcd
systemctl status etcd --no-pager

# ---- 4. Wait for etcd to become reachable ----
# After `systemctl restart` returns, etcd may still be starting up, so a single
# check can race. Retry for up to 15s. `endpoint status` needs no credentials.
echo "[4/4] Waiting for etcd at ${EP}"
if ! command -v etcdctl >/dev/null 2>&1; then
    echo "Error: etcdctl not found" >&2
    exit 1
fi

export ETCDCTL_API=3
unset ETCDCTL_USER ETCDCTL_ENDPOINTS ETCDCTL_USERNAME ETCDCTL_PASSWORD

ETCD_READY=false
for _ in $(seq 1 15); do
    if etcdctl --endpoints="${EP}" endpoint status >/dev/null 2>&1; then
        ETCD_READY=true
        break
    fi
    sleep 1
done
if ! $ETCD_READY; then
    echo "Error: etcd not reachable at ${EP} after 15s" >&2
    exit 1
fi
echo "      etcd ready (auth disabled)"

echo "----"
echo "Done. Verify with:"
echo "  etcdctl --endpoints=http://${LOCAL_IP}:2379 endpoint health"
