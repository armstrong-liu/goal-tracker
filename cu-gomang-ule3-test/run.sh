#!/bin/bash
#
# run.sh — orchestrates the cu-gomang functional test suite.
#
# Lifecycle (tone-cli calls each function individually; `bash run.sh` runs the
# full cycle locally):
#   setup()    -> provision env: install etcd, then install+configure gomang
#   run()      -> execute the case scripts under cases/
#   teardown() -> restore env: stop gomang (remove its unit), uninstall etcd
#   parse()    -> pass "case_name: pass/fail" lines through to result.json
#
# Each case wraps a verify_*.sh script and prints the tone-cli verdict line;
# verify detail is written to a per-case .log file (not stdout) so it cannot
# be mis-parsed as verdict lines.
#
# Local debug:
#   cd cu-gomang-test && bash run.sh
#   bash cases/execute_scripts_test.sh        # single case
#

SUITE_DIR=${TONE_BM_SUITE_DIR:-$(cd "$(dirname "$0")" && pwd)}

# Cases to run (one per verify_* script)
CASES=(
    execute_scripts_test
)

# tone calls setup()/run()/teardown() as SEPARATE processes, so setup failure
# cannot be signalled to run() via a shell variable. Use a marker file on the
# per-run working dir (falls back to the suite dir for local `bash run.sh`).
# NOTE: tone STILL calls run() after setup(); the marker makes run() emit
# "skip" for every case. setup() MUST return 0 on env failure — a non-zero
# exit makes runstatus/frontend "fail" even when all cases are skip.
SETUP_FAILED_MARKER="${TONE_BM_RUN_DIR:-$SUITE_DIR}/.setup_failed"

# ---------- setup: provision environment ----------
# setup_etcd.sh installs+configures etcd; must run before setup_gomang.sh.
setup() {
    rm -f "$SETUP_FAILED_MARKER"          # clear any stale marker from a prior run
    echo "=== setup install etcd ==="
    if ! bash "$SUITE_DIR/setup_etcd.sh"; then
        touch "$SETUP_FAILED_MARKER"
        echo "=== setup etcd failed; cases will be skipped ==="
        return 0
    fi

    echo "=== setup configure gomang ==="
    if ! bash "$SUITE_DIR/setup_gomang.sh"; then
        touch "$SETUP_FAILED_MARKER"
        echo "=== setup gomang failed; cases will be skipped ==="
        return 0
    fi
}

# ---------- run: execute cases ----------
run() {
    echo "=== run start ==="
    local c
    # If setup failed, don't run the cases (they'd all fail on a missing env and
    # obscure the real cause). Emit "skip" for each instead.
    if [ -f "$SETUP_FAILED_MARKER" ]; then
        echo "setup failed earlier; skipping all cases"
        for c in "${CASES[@]}"; do
            echo "${c}: skip"
        done
        echo "=== run finished (skipped) ==="
        return 0
    fi
    for c in "${CASES[@]}"; do
        echo "--- ${c} ---"
        bash "$SUITE_DIR/cases/${c}.sh"
    done
    echo "=== run finished ==="
}

# ---------- teardown: restore environment ----------
teardown() {
    echo "=== teardown restore environment ==="
    bash "$SUITE_DIR/teardown.sh"
    rm -f "$SETUP_FAILED_MARKER"
}

# ---------- parse: pass verdict lines through to result.json ----------
# tone feeds run()'s saved stdout to parse() on stdin; verdict lines are
# already in "<case>: pass|fail" form, so just forward them.
parse() {
    cat
}

# When invoked directly (no TONE_BM_RUN_DIR), run the full cycle.
# tone-cli sources this file with TONE_BM_RUN_DIR set and calls each function,
# so this block does not execute under tone.
if [ -z "${TONE_BM_RUN_DIR:-}" ]; then
    # Mirror tone: always call run() after setup(); run() decides skip-vs-execute
    # from the marker, so setup failure yields "<case>: skip" here too.
    setup || true
    run
    teardown
fi
