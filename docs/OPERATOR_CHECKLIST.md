# Operator Checklist: From Clone to Paper Bot to Live Trading

Use this as the overall checklist before trusting go-trader with real capital.

## Phase 1 — Local setup

- [ ] Install Go and `uv`.
- [ ] Run `uv sync` so `.venv/bin/python3` exists.
- [ ] Build the binary:

  ```bash
  VER=$(git describe --tags --always --dirty 2>/dev/null || echo dev)
  go -C scheduler build -ldflags "-X main.Version=$VER" -o ../go-trader .
  ```

## Phase 2 — Generate a paper-mode starter config

Pick one market profile:

```bash
bash scripts/quickstart-profile.sh stocks SPY,QQQ
bash scripts/quickstart-profile.sh currency 6E,6J
bash scripts/quickstart-profile.sh crypto BTC,ETH
```

Optional helper modes:

```bash
RUN_ONCE=1 bash scripts/quickstart-profile.sh stocks SPY,QQQ
START=1 bash scripts/quickstart-profile.sh currency 6E,6J
STRICT=1 bash scripts/quickstart-profile.sh crypto BTC,ETH
```

## Phase 3 — Preflight and smoke test

- [ ] Run human-readable preflight:

  ```bash
  ./go-trader --config scheduler/config.json --preflight
  ```

- [ ] Run machine-readable preflight:

  ```bash
  ./go-trader --config scheduler/config.json --preflight-json
  ```

- [ ] Run strict preflight before CI/deploy/live:

  ```bash
  ./go-trader --config scheduler/config.json --preflight-strict
  ```

- [ ] Run one scheduler cycle:

  ```bash
  ./go-trader --config scheduler/config.json --once
  ```

## Phase 4 — Run paper mode continuously

- [ ] Start the scheduler:

  ```bash
  ./go-trader --config scheduler/config.json
  ```

- [ ] Check health/status from another shell:

  ```bash
  curl -s localhost:8099/health
  curl -s localhost:8099/status | python3 -m json.tool
  ```

- [ ] Let paper mode run for 1–2 weeks before live trading.
- [ ] Confirm fills, PnL accounting, position state, and notifications look correct.

## Phase 5 — Prepare live trading

- [ ] Keep strategy size tiny at first.
- [ ] Set credentials only in environment variables or `.env` files that are not committed.
- [ ] Run strict preflight after adding live credentials.
- [ ] Confirm `scheduler/state.db` is present and backed up.
- [ ] Confirm status server remains loopback-only.

## Phase 6 — Ongoing operations

- [ ] Use `bash scripts/update.sh --restart` instead of ad-hoc rebuilds.
- [ ] Back up `scheduler/config.json` and `scheduler/state.db` before config changes.
- [ ] Keep one paper canary instance if you also run live.
- [ ] Re-run `--preflight-strict` after every config or dependency change.
