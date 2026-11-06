#!/usr/bin/env bash
# Self-test for ci/scripts/run_with_retry.sh. Local only: no network, no root,
# no external dependencies beyond bash + coreutils. Run from anywhere:
#   ci/scripts/tests/run_with_retry_test.sh
# Exits 0 when every check passes, 1 otherwise.

set -u

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WRAPPER="$(cd "$TEST_DIR/.." && pwd)/run_with_retry.sh"
WORK="$(mktemp -d "${TMPDIR:-/tmp}/run-with-retry-test.XXXXXX")"
trap 'rm -rf "$WORK"' EXIT

# Keep the suite fast: retries must not actually wait.
export NANOKVM_CI_BACKOFF=0

passed=0
failed=0

report_ok() {
    passed=$((passed + 1))
    printf 'ok %2d - %s\n' "$passed" "$1"
}

report_fail() {
    failed=$((failed + 1))
    printf 'not ok - %s\n' "$1" >&2
    if [ -n "${2:-}" ]; then
        printf '       %s\n' "$2" >&2
    fi
}

check_eq() {
    if [ "$2" = "$3" ]; then
        report_ok "$1"
    else
        report_fail "$1" "expected [$2], got [$3]"
    fi
}

check_contains() {
    case "$2" in
    *"$3"*) report_ok "$1" ;;
    *) report_fail "$1" "expected output to contain [$3]; got: $2" ;;
    esac
}

check_not_contains() {
    case "$2" in
    *"$3"*) report_fail "$1" "expected output NOT to contain [$3]" ;;
    *) report_ok "$1" ;;
    esac
}

# A command that fails with a given code for its first N invocations, then
# succeeds; the persisted counter doubles as the attempt counter.
cat >"$WORK/flaky.sh" <<'EOF'
#!/usr/bin/env bash
# flaky.sh <state-file> <fail-times> <fail-code>
set -u
state_file="$1"
fail_times="$2"
fail_code="$3"
n=0
if [ -f "$state_file" ]; then
    n="$(cat "$state_file")"
fi
n=$((n + 1))
printf '%s\n' "$n" >"$state_file"
if [ "$n" -le "$fail_times" ]; then
    exit "$fail_code"
fi
exit 0
EOF
chmod +x "$WORK/flaky.sh"

attempts_made() {
    if [ -f "$1" ]; then
        cat "$1"
    else
        printf '0'
    fi
}

# Run the wrapper with separated streams. Sets RUN_RC / OUT_TEXT / ERR_TEXT.
run_wrapper() {
    "$WRAPPER" "$@" >"$WORK/last.out" 2>"$WORK/last.err"
    RUN_RC=$?
    OUT_TEXT="$(cat "$WORK/last.out")"
    ERR_TEXT="$(cat "$WORK/last.err")"
}

# --- 1. clean pass -----------------------------------------------------------
run_wrapper --label t1 -- true
check_eq "first-attempt success exits 0" 0 "$RUN_RC"
check_contains "attempt line logged" "$ERR_TEXT" "attempt=1/3 exit=0"
check_not_contains "clean pass logs no RECOVERED" "$ERR_TEXT" "RECOVERED"
check_eq "wrapper itself writes nothing to stdout" "" "$OUT_TEXT"

# --- 2. recovery on attempt 2 ------------------------------------------------
rm -f "$WORK/r2.state" "$WORK/report.txt"
run_wrapper --label t2 --report "$WORK/report.txt" -- "$WORK/flaky.sh" "$WORK/r2.state" 1 1
check_eq "recovered run exits 0" 0 "$RUN_RC"
check_eq "child ran exactly twice" 2 "$(attempts_made "$WORK/r2.state")"
check_contains "RECOVERED line logged" "$ERR_TEXT" "RECOVERED on attempt=2/3"
if [ -f "$WORK/report.txt" ] && [ "$(wc -l <"$WORK/report.txt")" -eq 1 ]; then
    report_ok "report file got exactly one recovery line"
else
    report_fail "report file got exactly one recovery line"
fi
check_contains "report line carries the label" "$(cat "$WORK/report.txt")" "label=t2"

rm -f "$WORK/r2b.state" "$WORK/report3.txt"
run_wrapper --label t2b --report "$WORK/report3.txt" -- "$WORK/flaky.sh" "$WORK/r2b.state" 2 1
check_eq "recovery on attempt 3 exits 0" 0 "$RUN_RC"
check_eq "child ran exactly three times" 3 "$(attempts_made "$WORK/r2b.state")"
check_contains "RECOVERED logs attempt=3/3" "$ERR_TEXT" "RECOVERED on attempt=3/3"

# --- 3. persistent failure: exactly 3 attempts, child code propagates ---------
rm -f "$WORK/r3.state"
run_wrapper --label t3 --report "$WORK/report.txt" -- "$WORK/flaky.sh" "$WORK/r3.state" 99 1
check_eq "persistent failure exits with child code 1" 1 "$RUN_RC"
check_eq "exactly 3 attempts" 3 "$(attempts_made "$WORK/r3.state")"
check_contains "EXHAUSTED line logged" "$ERR_TEXT" "EXHAUSTED after 3/3 attempts (last exit=1)"

rm -f "$WORK/r4.state"
run_wrapper --label t4 -- "$WORK/flaky.sh" "$WORK/r4.state" 99 42
check_eq "child exit code 42 propagates" 42 "$RUN_RC"
check_eq "code 42 retried up to 3 attempts" 3 "$(attempts_made "$WORK/r4.state")"

# --- 4. deny-listed codes never retry -----------------------------------------
for code in 2 126 127 130 137 143; do
    rm -f "$WORK/deny.state"
    run_wrapper --label deny -- "$WORK/flaky.sh" "$WORK/deny.state" 99 "$code"
    check_eq "exit $code: exactly one attempt" 1 "$(attempts_made "$WORK/deny.state")"
    check_eq "exit $code: code propagated" "$code" "$RUN_RC"
done

# --- 5. exit-code allow/deny options ------------------------------------------
rm -f "$WORK/r5.state"
run_wrapper --label t5 --retry-exit-codes 1 -- "$WORK/flaky.sh" "$WORK/r5.state" 99 3
check_eq "--retry-exit-codes 1: exit 3 not retried" 1 "$(attempts_made "$WORK/r5.state")"
rm -f "$WORK/r5.state"
run_wrapper --label t5 --retry-exit-codes 1 -- "$WORK/flaky.sh" "$WORK/r5.state" 99 1
check_eq "--retry-exit-codes 1: exit 1 retried 3 times" 3 "$(attempts_made "$WORK/r5.state")"
rm -f "$WORK/r5.state"
run_wrapper --label t5 --no-retry-exit-codes 1 -- "$WORK/flaky.sh" "$WORK/r5.state" 99 1
check_eq "--no-retry-exit-codes 1: exit 1 not retried" 1 "$(attempts_made "$WORK/r5.state")"

# --- 6. nested invocation guard -----------------------------------------------
rm -f "$WORK/r6.state"
NESTED_ERR="$(
    (
        export NANOKVM_CI_RETRY_DEPTH=1
        "$WRAPPER" --label t6 -- "$WORK/flaky.sh" "$WORK/r6.state" 99 1
    ) 2>&1
)"
NESTED_RC=$?
check_eq "nested run exits with child code" 1 "$NESTED_RC"
check_eq "nested run executes exactly once" 1 "$(attempts_made "$WORK/r6.state")"
check_contains "nested refusal logged" "$NESTED_ERR" "nested invocation refused"

# --- 7. usage errors and clamping ----------------------------------------------
"$WRAPPER" >/dev/null 2>&1
check_eq "no arguments is a usage error" 2 "$?"
"$WRAPPER" -- >/dev/null 2>&1
check_eq "missing command after -- is a usage error" 2 "$?"
"$WRAPPER" true >/dev/null 2>&1
check_eq "missing -- separator is a usage error" 2 "$?"
"$WRAPPER" --attempts abc -- true >/dev/null 2>&1
check_eq "non-numeric --attempts is a usage error" 2 "$?"
"$WRAPPER" --help >/dev/null 2>&1
check_eq "--help exits 0" 0 "$?"

rm -f "$WORK/r7.state"
run_wrapper --label t7 --attempts 9 -- "$WORK/flaky.sh" "$WORK/r7.state" 99 1
check_eq "--attempts 9 clamps to 3 attempts" 3 "$(attempts_made "$WORK/r7.state")"
check_contains "clamp is logged" "$ERR_TEXT" "clamped to 3"

rm -f "$WORK/r7b.state"
run_wrapper --label t7b --attempts 1 -- "$WORK/flaky.sh" "$WORK/r7b.state" 99 1
check_eq "--attempts 1 runs exactly once" 1 "$(attempts_made "$WORK/r7b.state")"

rm -f "$WORK/r7c.state"
(
    export NANOKVM_CI_ATTEMPTS=1
    "$WRAPPER" --label t7c -- "$WORK/flaky.sh" "$WORK/r7c.state" 99 1 >/dev/null 2>&1
)
check_eq "NANOKVM_CI_ATTEMPTS=1 runs exactly once" 1 "$(attempts_made "$WORK/r7c.state")"

# --- 8. command preflight -------------------------------------------------------
run_wrapper --label t8 -- definitely-not-a-real-command-xyz
check_eq "missing command exits 127" 127 "$RUN_RC"
check_contains "missing command is logged" "$ERR_TEXT" "command not found"

# --- 9. --cleanup-rm --------------------------------------------------------------
mkdir -p "$WORK/cleanme/sub"
touch "$WORK/cleanme/sub/marker"
rm -f "$WORK/r9.state"
(
    cd "$WORK" && "$WRAPPER" --label t9 --cleanup-rm cleanme -- \
        "$WORK/flaky.sh" "$WORK/r9.state" 1 1 >/dev/null 2>&1
)
check_eq "cleanup run recovers" 0 "$?"
if [ ! -e "$WORK/cleanme" ]; then
    report_ok "cleanup path removed before the retry"
else
    report_fail "cleanup path removed before the retry"
fi

"$WRAPPER" --cleanup-rm / -- true >/dev/null 2>&1
check_eq "--cleanup-rm / rejected with exit 2" 2 "$?"
"$WRAPPER" --cleanup-rm "" -- true >/dev/null 2>&1
check_eq "--cleanup-rm '' rejected with exit 2" 2 "$?"
(
    cd "$WORK" && "$WRAPPER" --cleanup-rm ../escape -- true >/dev/null 2>&1
)
check_eq "--cleanup-rm escaping the root rejected" 2 "$?"
(
    cd "$WORK" && "$WRAPPER" --cleanup-rm no-such-dir -- true >/dev/null 2>&1
)
check_eq "nonexistent in-bounds cleanup path is a no-op" 0 "$?"

# Boundary follows CI_PROJECT_DIR, not the cwd, when the variable is set.
mkdir -p "$WORK/proj/sub" "$WORK/proj/cleanme2"
rm -f "$WORK/r9b.state"
(
    cd "$WORK/proj/sub" && CI_PROJECT_DIR="$WORK/proj" \
        "$WRAPPER" --label t9b --cleanup-rm ../cleanme2 -- \
        "$WORK/flaky.sh" "$WORK/r9b.state" 1 1 >/dev/null 2>&1
)
check_eq "cleanup above cwd but inside CI_PROJECT_DIR allowed" 0 "$?"
if [ ! -e "$WORK/proj/cleanme2" ]; then
    report_ok "CI_PROJECT_DIR-relative cleanup removed the path"
else
    report_fail "CI_PROJECT_DIR-relative cleanup removed the path"
fi
(
    cd "$WORK/proj/sub" && CI_PROJECT_DIR="$WORK/proj" \
        "$WRAPPER" --cleanup-rm ../../outside -- true >/dev/null 2>&1
)
check_eq "cleanup escaping CI_PROJECT_DIR rejected" 2 "$?"

# --- 10. label default ------------------------------------------------------------
rm -f "$WORK/r10.state"
run_wrapper -- "$WORK/flaky.sh" "$WORK/r10.state" 0 1
check_contains "default label is the command basename" "$ERR_TEXT" "label=flaky.sh"

# --- 11. default report path ---------------------------------------------------------
rm -f "$WORK/r11.state" "$WORK/.ci-retry-report.txt"
(
    unset CI_PROJECT_DIR
    cd "$WORK" && "$WRAPPER" --label t11 -- "$WORK/flaky.sh" "$WORK/r11.state" 1 1 >/dev/null 2>&1
)
if [ -f "$WORK/.ci-retry-report.txt" ]; then
    report_ok "default report path is \$PWD/.ci-retry-report.txt"
else
    report_fail "default report path is \$PWD/.ci-retry-report.txt"
fi

# --- 12. stream passthrough -------------------------------------------------------------
run_wrapper -- bash -c 'echo to-stdout; echo to-stderr >&2'
check_contains "child stdout is passed through" "$OUT_TEXT" "to-stdout"
check_contains "child stderr is passed through" "$ERR_TEXT" "to-stderr"
check_not_contains "wrapper metadata stays off stdout" "$OUT_TEXT" "[ci-retry]"

# --- 13. signals: forward and stop, never retry --------------------------------------------
start="$SECONDS"
"$WRAPPER" --label sigterm -- sleep 30 >/dev/null 2>"$WORK/sigterm.err" &
sig_pid=$!
sleep 1
kill -TERM "$sig_pid"
wait "$sig_pid"
SIG_RC=$?
check_eq "SIGTERM exits 143" 143 "$SIG_RC"
if [ $((SECONDS - start)) -lt 10 ]; then
    report_ok "SIGTERM interrupts the attempt promptly"
else
    report_fail "SIGTERM interrupts the attempt promptly"
fi
check_not_contains "no retry after SIGTERM" "$(cat "$WORK/sigterm.err")" "EXHAUSTED"

# SIGINT is ignored at entry for background jobs of a non-interactive shell,
# so the wrapper must run in the foreground here (as it does under GitLab) and
# the signal must come from a separate watcher process.
sigint_err="$WORK/sigint.err"
(
    sleep 1
    pkill -INT -f 'label sigint' 2>/dev/null
) &
"$WRAPPER" --label sigint -- sleep 30 >/dev/null 2>"$sigint_err"
check_eq "SIGINT exits 130" 130 "$?"
check_not_contains "no retry after SIGINT" "$(cat "$sigint_err")" "EXHAUSTED"

# --- summary ---------------------------------------------------------------------------------
printf '%d passed, %d failed\n' "$passed" "$failed"
if [ "$failed" -gt 0 ]; then
    exit 1
fi
exit 0
