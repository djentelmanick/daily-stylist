#!/usr/bin/env bash
# Туннель и все процессы приложения в одном терминале, адрес туннеля подставляется сам.
set -euo pipefail

cd "$(dirname "$0")/.."

ngrok_api=http://127.0.0.1:4040/api/tunnels
pids=()

kill_tree() {
    local pid=$1 child
    for child in $(pgrep -P "$pid" 2>/dev/null || true); do
        kill_tree "$child"
    done
    kill "$pid" 2>/dev/null || true
}

cleanup() {
    trap - INT TERM EXIT
    for pid in "${pids[@]}"; do
        kill_tree "$pid"
    done
    wait 2>/dev/null || true
}
trap cleanup INT TERM EXIT

start() {
    local name=$1
    shift
    ("$@" 2>&1 | sed -u "s/^/[$name] /") &
    pids+=($!)
}

tunnel_url() {
    curl -s --max-time 2 "$ngrok_api" 2>/dev/null |
        grep -o '"public_url":"https://[^"]*"' | head -1 | cut -d'"' -f4 || true
}

if [ -n "$(tunnel_url)" ]; then
    echo "[tun] туннель уже работает"
else
    start tun ngrok http 5173 --log stdout --log-level warn ${NGROK_DOMAIN:+--domain "$NGROK_DOMAIN"}
fi

url=""
for _ in $(seq 30); do
    url=$(tunnel_url)
    if [ -n "$url" ]; then
        break
    fi
    sleep 0.5
done
if [ -z "$url" ]; then
    echo "туннель не поднялся, ngrok не ответил за 15 секунд" >&2
    exit 1
fi

# Процессы читают .env, но godotenv не перебивает то, что уже задано в окружении.
export TELEGRAM_WEBHOOK_BASE_URL="$url"
export S3_PUBLIC_URL="$url"
echo "[tun] $url"

start bot    sh -c 'cd backend && exec go run ./cmd/bot'
start sched  sh -c 'cd backend && exec go run ./cmd/scheduler'
start sender sh -c 'cd backend && exec go run ./cmd/sender'
start front  sh -c 'cd frontend && exec npm run dev'

wait
