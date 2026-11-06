#!/usr/bin/env sh
set -eu

HOST="${HOST:-127.0.0.1}"
PORT="${PORT:-65501}"
TOKEN="${TOKEN:-}"
INTERVAL="${INTERVAL:-0.35}"
LOOPS="${LOOPS:-3}"
URL="${URL:-http://${HOST}:${PORT}/api/v1/hid/state}"

if ! command -v curl >/dev/null 2>&1; then
    printf '%s\n' 'curl is required but was not found' >&2
    exit 127
fi

post_state() {
    payload="$1"
    printf 'POST %s %s\n' "${URL}" "${payload}"
    if [ -n "${TOKEN}" ]; then
        curl -fsS -X POST "${URL}" \
            -H 'Content-Type: application/json' \
            -H "Authorization: Bearer ${TOKEN}" \
            -d "${payload}"
    else
        curl -fsS -X POST "${URL}" \
            -H 'Content-Type: application/json' \
            -d "${payload}"
    fi
    printf '\n'
}

i=1
while [ "${i}" -le "${LOOPS}" ]; do
    post_state '{"mouse_active":true,"keyboard_special":"ctrl","keyboard_normal":"a"}'
    sleep "${INTERVAL}"
    post_state '{"mouse_active":true,"keyboard_special":"shift","keyboard_normal":"B"}'
    sleep "${INTERVAL}"
    post_state '{"mouse_active":false,"keyboard_special":"tab","keyboard_normal":""}'
    sleep "${INTERVAL}"
    post_state '{"mouse_active":true,"keyboard_special":"alt","keyboard_normal":"z"}'
    sleep "${INTERVAL}"
    i=$((i + 1))
done

post_state '{"mouse_active":false,"keyboard_special":"","keyboard_normal":""}'
