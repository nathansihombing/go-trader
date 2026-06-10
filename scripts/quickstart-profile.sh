#!/usr/bin/env bash
# Generate a profile-based paper starter config and run the built-in preflight.
# Examples:
#   bash scripts/quickstart-profile.sh stocks SPY,QQQ
#   bash scripts/quickstart-profile.sh currency 6E,6J
#   bash scripts/quickstart-profile.sh crypto BTC,ETH
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: bash scripts/quickstart-profile.sh <profile> [symbols] [output]

Profiles:
  stocks      Robinhood stock-options starter. Symbols default to SPY,QQQ.
  currency    Currency/FX futures starter. Symbols default to 6E,6J.
  fx|forex    Alias for currency.
  crypto      Crypto spot starter. Symbols default to BTC,ETH.

Arguments:
  symbols     Comma-separated symbols for the chosen profile.
  output      Config path; default scheduler/config.json.

Environment:
  SKIP_UV=1          Do not run uv sync when .venv is missing.
  SKIP_BUILD=1       Do not rebuild ./go-trader first.
  STRICT=1           Run --preflight-strict instead of --preflight.
  RUN_ONCE=1         Run one scheduler cycle after preflight.
  START=1            Start go-trader in the background after preflight.
  FORCE=1            Do not back up an existing output config first.
  STATUS_PORT=8099   Port to poll when START=1.
  HEALTH_TIMEOUT=30  Seconds to wait for /health when START=1.
  GO_BIN=/path/go    Override Go binary.
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

profile="${1:-stocks}"
symbols="${2:-}"
output="${3:-scheduler/config.json}"

case "$profile" in
  stocks|stock|equities|equity)
    init_profile="stocks"
    symbol_flag="--stock-symbols"
    symbols="${symbols:-SPY,QQQ}"
    ;;
  currency|currencies|fx|forex)
    init_profile="currency"
    symbol_flag="--currency-symbols"
    symbols="${symbols:-6E,6J}"
    ;;
  crypto)
    init_profile="crypto"
    symbol_flag="--assets"
    symbols="${symbols:-BTC,ETH}"
    ;;
  *)
    echo "unknown profile: $profile" >&2
    usage >&2
    exit 2
    ;;
esac

repo_root=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
cd "$repo_root"

if [[ "${SKIP_UV:-0}" != "1" && ! -x .venv/bin/python3 ]]; then
  if command -v uv >/dev/null 2>&1; then
    uv sync
  else
    cat >&2 <<'UV_WARN'
warning: .venv/bin/python3 is missing and uv is not on PATH.
         Run `uv sync` before RUN_ONCE=1 or long-running scheduler use.
UV_WARN
  fi
fi

if [[ "${SKIP_BUILD:-0}" != "1" ]]; then
  go_bin="${GO_BIN:-go}"
  if ! command -v "$go_bin" >/dev/null 2>&1; then
    if [[ -x /opt/homebrew/bin/go ]]; then
      go_bin=/opt/homebrew/bin/go
    elif [[ -x /usr/local/go/bin/go ]]; then
      go_bin=/usr/local/go/bin/go
    else
      echo "go binary not found; set GO_BIN=/path/to/go" >&2
      exit 1
    fi
  fi
  ver=$(git describe --tags --always --dirty 2>/dev/null || echo dev)
  "$go_bin" -C scheduler build -ldflags "-X main.Version=$ver" -o ../go-trader .
fi

mkdir -p "$(dirname "$output")"
if [[ -f "$output" && "${FORCE:-0}" != "1" ]]; then
  backup="$output.bak.$(date +%Y%m%d%H%M%S)"
  cp "$output" "$backup"
  echo "Existing config backed up to $backup"
fi
./go-trader init --profile "$init_profile" "$symbol_flag" "$symbols" --output "$output"

preflight_flag="--preflight"
if [[ "${STRICT:-0}" == "1" ]]; then
  preflight_flag="--preflight-strict"
fi
./go-trader --config "$output" "$preflight_flag"

if [[ "${RUN_ONCE:-0}" == "1" ]]; then
  ./go-trader --config "$output" --once
fi

if [[ "${START:-0}" == "1" ]]; then
  mkdir -p logs
  pidfile="${GO_TRADER_PIDFILE:-./go-trader.pid}"
  log_file="${QUICKSTART_LOG:-logs/quickstart.log}"
  if [[ -f "$pidfile" ]] && kill -0 "$(cat "$pidfile")" 2>/dev/null; then
    echo "go-trader already appears to be running with PID $(cat "$pidfile") ($pidfile)"
  else
    nohup ./go-trader --config "$output" >"$log_file" 2>&1 &
    echo $! >"$pidfile"
    echo "Started go-trader PID $(cat "$pidfile") (log: $log_file)"
  fi
  if command -v curl >/dev/null 2>&1; then
    status_port="${STATUS_PORT:-8099}"
    deadline=$((SECONDS + ${HEALTH_TIMEOUT:-30}))
    until curl -fsS "http://127.0.0.1:${status_port}/health" >/dev/null 2>&1; do
      if (( SECONDS >= deadline )); then
        echo "warning: /health did not become ready on port $status_port before timeout" >&2
        echo "         Check $log_file for details." >&2
        break
      fi
      sleep 1
    done
    if (( SECONDS < deadline )); then
      echo "Health check OK: http://127.0.0.1:${status_port}/health"
    fi
  fi
fi

cat <<EOF_DONE

Quickstart config ready: $output

Next commands:
  ./go-trader --config $output --preflight-json
  ./go-trader --config $output --once
  START=1 bash scripts/quickstart-profile.sh $init_profile $symbols $output
  ./go-trader --config $output

Then inspect:
  curl -s localhost:8099/health
  curl -s localhost:8099/status | python3 -m json.tool
EOF_DONE
