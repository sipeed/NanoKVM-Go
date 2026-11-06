#!/usr/bin/env bash
# run_with_retry.sh — run one CI command with bounded retries.
#
# Purpose: CI containers lose network intermittently; a transient fetch failure
# (apt/pip/npm/pnpm/conan/go modules/git clone) should not instantly fail the
# whole job. This wrapper re-runs the command up to <attempts> total attempts
# (default 3) with jittered backoff and fails only when every attempt fails.
#
# Wrap ONLY commands that are transient-failing, idempotent, and not the
# correctness verdict of a job. Never wrap tests, lint, audits, or anything
# with side effects (uploads, pushes, releases).
#
# Usage:
#   run_with_retry.sh [OPTIONS] -- <command> [args...]
#
# Options:
#   --label LABEL              Label for log lines and the report file
#                              (default: basename of the command).
#   --attempts N               Total attempts, clamped to 1..3 (default 3,
#                              env fallback NANOKVM_CI_ATTEMPTS).
#   --backoff-seconds N        Base backoff in seconds (default 5, env fallback
#                              NANOKVM_CI_BACKOFF). Attempt 2 waits ~N seconds,
#                              attempt 3 ~3N, each +/-20% jitter, capped at 30.
#   --retry-exit-codes SPEC    Comma-separated allow-list of exit codes to
#                              retry (default: every code not on the deny list).
#   --no-retry-exit-codes SPEC Comma-separated extra codes to never retry.
#   --cleanup-rm PATH          Remove PATH (rm -rf) before attempts 2..N.
#                              Repeatable. PATH resolves against the current
#                              directory and must land strictly inside
#                              ${CI_PROJECT_DIR:-$PWD}; invalid -> exit 2.
#   --report FILE              Recovery report file
#                              (default ${CI_PROJECT_DIR:-$PWD}/.ci-retry-report.txt).
#   -h, --help                 Print this help and exit 0.
#   --                         Mandatory separator before the command.
#
# Never retried (deny list): 0, 2, 126, 127, 130, 137, 143.
#   2 = usage error, 126/127 = not executable / not found, 130 = SIGINT,
#   137 = SIGKILL (OOM), 143 = SIGTERM (cancellation). Retrying those is either
#   useless or actively wrong.
#
# Exit status: the last child exit code (success or exhaustion); 2 for wrapper
# usage errors; 130/143 when terminated by SIGINT/SIGTERM.
#
# Nesting: the child runs with NANOKVM_CI_RETRY_DEPTH incremented. Entering the
# wrapper with depth >= 1 runs the command exactly once — retry layers must
# never multiply.
#
# Visibility: every attempt logs one '[ci-retry]' metadata line; a recovery
# logs a greppable RECOVERED line and appends to the report file. Recovery
# never fails the job. The child's stdout/stderr are inherited untouched.

set -u

readonly DEFAULT_ATTEMPTS=3
readonly DEFAULT_BACKOFF_SECONDS=5
readonly MAX_ATTEMPTS=3
readonly MAX_BACKOFF_SECONDS=30
# Space-padded list for exact-match lookups: " code ".
readonly DENY_EXIT_CODES=" 0 2 126 127 130 137 143 "

label=""
attempts="${NANOKVM_CI_ATTEMPTS:-$DEFAULT_ATTEMPTS}"
backoff_seconds="${NANOKVM_CI_BACKOFF:-$DEFAULT_BACKOFF_SECONDS}"
retry_exit_codes=""
extra_deny_exit_codes=""
report_file=""
cleanup_paths=()

log() {
    printf '[ci-retry] %s\n' "$*" >&2
}

usage() {
    # Print the header comment block: from line 2 up to the first non-comment
    # line, stripped of the leading '#'. Self-referential on purpose so the
    # help text can never drift from the file's own documentation.
    tail -n +2 "$0" | sed -n '/^#/!q; p' | sed -e 's/^# \{0,1\}//'
}

usage_error() {
    log "usage error: $*"
    exit 2
}

is_natural_number() {
    case "$1" in
    '' | *[!0-9]*) return 1 ;;
    *) return 0 ;;
    esac
}

while [ "$#" -gt 0 ]; do
    case "$1" in
    --label)
        [ "$#" -ge 2 ] || usage_error "--label needs a value"
        label="$2"
        shift 2
        ;;
    --attempts)
        [ "$#" -ge 2 ] || usage_error "--attempts needs a value"
        attempts="$2"
        shift 2
        ;;
    --backoff-seconds)
        [ "$#" -ge 2 ] || usage_error "--backoff-seconds needs a value"
        backoff_seconds="$2"
        shift 2
        ;;
    --retry-exit-codes)
        [ "$#" -ge 2 ] || usage_error "--retry-exit-codes needs a value"
        retry_exit_codes="$2"
        shift 2
        ;;
    --no-retry-exit-codes)
        [ "$#" -ge 2 ] || usage_error "--no-retry-exit-codes needs a value"
        extra_deny_exit_codes="$2"
        shift 2
        ;;
    --cleanup-rm)
        [ "$#" -ge 2 ] || usage_error "--cleanup-rm needs a value"
        cleanup_paths+=("$2")
        shift 2
        ;;
    --report)
        [ "$#" -ge 2 ] || usage_error "--report needs a value"
        report_file="$2"
        shift 2
        ;;
    -h | --help)
        usage
        exit 0
        ;;
    --)
        shift
        break
        ;;
    *)
        usage_error "unknown option '$1' (the command must follow a literal --)"
        ;;
    esac
done

[ "$#" -gt 0 ] || usage_error "missing command after --"

is_natural_number "$attempts" || usage_error "--attempts must be a number, got '$attempts'"
if [ "$attempts" -lt 1 ]; then
    log "attempts clamped to 1 (requested $attempts)"
    attempts=1
elif [ "$attempts" -gt "$MAX_ATTEMPTS" ]; then
    log "attempts clamped to $MAX_ATTEMPTS (requested $attempts)"
    attempts="$MAX_ATTEMPTS"
fi

is_natural_number "$backoff_seconds" || usage_error "--backoff-seconds must be a number, got '$backoff_seconds'"

if [ -z "$label" ]; then
    label="$(basename -- "$1")"
fi
if [ -z "$report_file" ]; then
    report_file="${CI_PROJECT_DIR:-$PWD}/.ci-retry-report.txt"
fi
cmd_display="$*"

# A cleanup path must resolve strictly inside the project root so a mistyped
# call site can never delete anything outside the checkout.
resolve_cleanup_path() {
    local path="$1" root resolved
    [ -n "$path" ] || return 1
    [ "$path" != "/" ] || return 1
    root="${CI_PROJECT_DIR:-$PWD}"
    case "$path" in
    /*) resolved="$path" ;;
    *) resolved="$PWD/$path" ;;
    esac
    # type -P: the CI policy guard forbids the plain 'command' lookup flag
    # (its -v token collides with the volume-mount pattern), so probe PATH
    # entries only.
    if type -P realpath >/dev/null 2>&1; then
        resolved="$(realpath -m -- "$resolved")" || return 1
        root="$(realpath -m -- "$root")" || return 1
    else
        # Without realpath, parent references cannot be resolved safely.
        case "$path" in
        *..*) return 1 ;;
        esac
    fi
    [ "$resolved" != "$root" ] || return 1
    case "$resolved/" in
    "$root"/*)
        printf '%s' "$resolved"
        return 0
        ;;
    *) return 1 ;;
    esac
}

resolved_cleanup_paths=()
if [ "${#cleanup_paths[@]}" -gt 0 ]; then
    for path in "${cleanup_paths[@]}"; do
        resolved="$(resolve_cleanup_path "$path")" ||
            usage_error "--cleanup-rm path escapes \${CI_PROJECT_DIR:-\$PWD}: '$path'"
        resolved_cleanup_paths+=("$resolved")
    done
fi

preflight_command() {
    local cmd="$1"
    case "$cmd" in
    */*)
        if [ ! -e "$cmd" ]; then
            log "command not found: $cmd"
            exit 127
        fi
        if [ ! -x "$cmd" ]; then
            log "command not executable: $cmd"
            exit 126
        fi
        ;;
    *)
        if ! type -P "$cmd" >/dev/null 2>&1; then
            log "command not found: $cmd"
            exit 127
        fi
        ;;
    esac
}

child_pid=""
terminate() {
    # Forward cancellation to the child and exit without further attempts.
    local code="$1"
    trap '' TERM INT
    if [ -n "$child_pid" ]; then
        kill -TERM "$child_pid" 2>/dev/null
        wait "$child_pid" 2>/dev/null
    fi
    exit "$code"
}
trap 'terminate 143' TERM
trap 'terminate 130' INT

run_child() {
    "$@" &
    child_pid=$!
    if wait "$child_pid"; then
        rc=0
    else
        rc=$?
    fi
    child_pid=""
    return "$rc"
}

append_report() {
    [ -n "$report_file" ] || return 0
    printf '%s label=%s recovered_on_attempt=%s/%s cmd=%s\n' \
        "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$label" "$1" "$attempts" "$cmd_display" \
        >>"$report_file" 2>/dev/null || true
}

should_retry() {
    local code="$1"
    case "$DENY_EXIT_CODES" in
    *" $code "*) return 1 ;;
    esac
    if [ -n "$extra_deny_exit_codes" ]; then
        case ",$extra_deny_exit_codes," in
        *",$code,"*) return 1 ;;
        esac
    fi
    if [ -n "$retry_exit_codes" ]; then
        case ",$retry_exit_codes," in
        *",$code,"*) return 0 ;;
        *) return 1 ;;
        esac
    fi
    return 0
}

backoff_for_attempt() {
    # $1 = attempt that just failed. Base, then triple, jitter +/-20%, and cap
    # last so the cap bounds the effective wait rather than its pre-jitter base.
    local delay
    if [ "$1" -le 1 ]; then
        delay="$backoff_seconds"
    else
        delay=$((backoff_seconds * 3))
    fi
    if [ "$delay" -gt 0 ]; then
        delay=$((delay * (80 + RANDOM % 41) / 100))
    fi
    if [ "$delay" -gt "$MAX_BACKOFF_SECONDS" ]; then
        delay="$MAX_BACKOFF_SECONDS"
    fi
    printf '%s' "$delay"
}

run_cleanups() {
    local path
    if [ "${#resolved_cleanup_paths[@]}" -eq 0 ]; then
        return 0
    fi
    for path in "${resolved_cleanup_paths[@]}"; do
        log "label=$label cleanup before attempt $attempt: rm -rf $path"
        rm -rf -- "$path" || log "cleanup failed for $path (continuing anyway)"
    done
}

depth="${NANOKVM_CI_RETRY_DEPTH:-0}"
is_natural_number "$depth" || depth=0

preflight_command "$1"

if [ "$depth" -ge 1 ]; then
    log "label=$label nested invocation refused (NANOKVM_CI_RETRY_DEPTH=$depth); running once"
    run_child "$@"
    exit "$rc"
fi

export NANOKVM_CI_RETRY_DEPTH=$((depth + 1))

attempt=1
while :; do
    if [ "$attempt" -gt 1 ]; then
        run_cleanups
    fi

    start="$SECONDS"
    run_child "$@"
    duration=$((SECONDS - start))
    log "label=$label attempt=$attempt/$attempts exit=$rc duration=${duration}s cmd=$cmd_display"

    if [ "$rc" -eq 0 ]; then
        if [ "$attempt" -gt 1 ]; then
            log "label=$label RECOVERED on attempt=$attempt/$attempts -- flaky, needs triage"
            append_report "$attempt"
        fi
        exit 0
    fi

    if ! should_retry "$rc"; then
        log "label=$label exit=$rc is not retryable"
        exit "$rc"
    fi

    if [ "$attempt" -ge "$attempts" ]; then
        log "label=$label EXHAUSTED after $attempt/$attempts attempts (last exit=$rc)"
        exit "$rc"
    fi

    delay="$(backoff_for_attempt "$attempt")"
    log "label=$label retrying in ${delay}s"
    if [ "$delay" -gt 0 ]; then
        # Background + wait so a TERM during backoff is handled by the trap
        # immediately instead of after the sleep finishes.
        sleep "$delay" &
        child_pid=$!
        wait "$child_pid" 2>/dev/null
        child_pid=""
    fi
    attempt=$((attempt + 1))
done
