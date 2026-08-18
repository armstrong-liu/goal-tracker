#!/bin/bash
#
# install.sh — called by `tone install`.
#
# This suite is pure shell scripts (no external source to fetch or compile).
# Environment provisioning (install etcd + configure gomang) is done in
# run.sh setup() so each `tone run` is self-contained.
#

# No external git/web sources to fetch
GIT_URL=""
WEB_URL=""
BRANCH=""

# No packages to install here; etcd is installed by setup_etcd.sh at run time
DEP_PKG_LIST=""

build() {
    echo "[cu-gomang-test] script-only suite, nothing to build"
}

install() {
    echo "[cu-gomang-test] nothing to install; env setup runs in run.sh setup()"
}
