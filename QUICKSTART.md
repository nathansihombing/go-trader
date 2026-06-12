# Quickstart: Get go-trader Running

This is the shortest safe path from a fresh checkout to a working paper-mode bot.


## One-command helper

You can generate a starter config and run preflight in one command:

```bash
bash scripts/quickstart-profile.sh stocks SPY,QQQ
bash scripts/quickstart-profile.sh currency 6E,6J
bash scripts/quickstart-profile.sh crypto BTC,ETH
START=1 bash scripts/quickstart-profile.sh stocks SPY,QQQ
```

The helper runs `uv sync` automatically when `.venv/bin/python3` is missing (set `SKIP_UV=1` to skip). Set `RUN_ONCE=1` to also run one scheduler cycle after preflight, `START=1` to launch the bot in the background and poll `/health`, or `STRICT=1` to make warnings fail the preflight gate. Existing configs are backed up automatically unless `FORCE=1`. Use `STATUS_PORT=8100` if port `8099` is already in use.

## 1) Install dependencies and build

```bash
uv sync
VER=$(git describe --tags --always --dirty 2>/dev/null || echo dev)
go -C scheduler build -ldflags "-X main.Version=$VER" -o ../go-trader .
```

## 2) Create a starter config

Pick **one** profile first. These commands generate paper/signal-first configs by default.

### Stocks starter

```bash
./go-trader init --profile stocks --stock-symbols SPY,QQQ --output scheduler/config.json
```

This creates Robinhood stock-options strategies. Live trading later requires Robinhood credentials, but paper/signal testing does not.

### Currency / FX starter

```bash
./go-trader init --profile currency --currency-symbols 6E,6J --output scheduler/config.json
```

This creates TopStep futures strategies for major currency futures. Live trading later requires TopStep credentials, but paper testing does not.

### Crypto starter

```bash
./go-trader init --profile crypto --assets BTC,ETH --output scheduler/config.json
```

This creates crypto spot strategies.

## 3) Audit the config

```bash
./go-trader --config scheduler/config.json --preflight
./go-trader --config scheduler/config.json --preflight-json
```

Use strict mode before deploys or in CI:

```bash
./go-trader --config scheduler/config.json --preflight-strict
```

## 4) Run one paper cycle

```bash
./go-trader --config scheduler/config.json --once
```

If this succeeds, the config loads, strategy scripts can run, and state persistence works.

## 5) Start the local status server

```bash
./go-trader --config scheduler/config.json
```

Then in another shell:

```bash
curl -s localhost:8099/health
curl -s localhost:8099/status | python3 -m json.tool
```

If you used `START=1`, stop the background process with:

```bash
kill "$(cat ./go-trader.pid)"
```

The server binds to loopback only. Keep it that way and use VPN/reverse proxy if you need remote access.

## 6) Only then consider live mode

Before live trading:

- run paper mode for at least 1–2 weeks;
- verify alerts and `/status` output;
- keep strategy capital tiny at first;
- configure exchange credentials through environment variables, not committed files;
- run `./go-trader --config scheduler/config.json --preflight-strict` cleanly.

See [START_SAFE.md](START_SAFE.md) for the full go-live checklist.
