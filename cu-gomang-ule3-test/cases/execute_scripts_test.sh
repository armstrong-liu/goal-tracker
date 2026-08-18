#!/bin/bash
#
# Case execute_scripts_test:
#   Push a shell-script task to gomang via etcd and verify the result.
#   Wraps verify_execute_scripts.sh.
#
# Output: "execute_scripts_test: pass|fail" (tone-cli verdict line)
# Verify detail is written to a per-case log file (see the line printed below),
# kept out of stdout so it cannot be mis-parsed as verdict lines.
#
# Debug: bash cases/execute_scripts_test.sh
#

CASE_NAME="execute_scripts_test"
SUITE_DIR=${TONE_BM_SUITE_DIR:-$(cd "$(dirname "$0")/.." && pwd)}
RESULT_DIR=${TONE_CURRENT_RESULT_DIR:-/tmp}
LOG="${RESULT_DIR}/${CASE_NAME}.log"

if bash "$SUITE_DIR/verify_execute_scripts.sh" >"$LOG" 2>&1; then
    echo "${CASE_NAME}: pass"
else
    echo "${CASE_NAME}: fail"
fi
echo "  detail log -> ${LOG}"
