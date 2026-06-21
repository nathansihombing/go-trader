# Start Safely: Pre-Flight Checklist for go-trader

If you're about to use go-trader with real money, do these in order.

## 1) Run in paper mode first (minimum 1–2 weeks)

- Use paper args for every strategy.
- Keep this running continuously and inspect `/status` and `/history` daily.
- Do **not** enable live API keys yet.

## 2) Start with one strategy and one asset

- Begin with a single strategy on a single symbol (for example BTC only).
- Avoid multi-strategy same-coin setups until you have seen clean behavior for several days.

## 3) Verify risk rails before any live order

Set and verify these config controls:

- `portfolio_risk.max_drawdown_pct`
- `portfolio_risk.max_notional_usd`
- strategy `max_drawdown_pct`
- perps stop-loss configuration (`stop_loss_*`/`trailing_stop_*`)

Then run one cycle and validate logs/status:

```bash
./go-trader --config scheduler/config.json --once
curl -s localhost:8099/status | python3 -m json.tool
```

## 4) Enforce staged capital rollout

Use a strict rollout plan:

1. Week 1 live: tiny size only (e.g. 1–5% of planned deployment).
2. Week 2+: increase only if fills, PnL accounting, and alerts are all correct.
3. Scale down immediately on anomalies (missing fills, stale prices, repeated rejects).

## 5) Lock in operational safety

- Keep status server loopback-only (default) and access remotely through VPN/reverse proxy.
- Enable owner notifications in Discord/Telegram.
- Run `uv sync`, then `./go-trader --config scheduler/config.json --preflight-strict`, and resolve missing scripts, missing `.venv/bin/python3`, and missing live credential environment variables before switching any strategy to live.
- Keep backups of `scheduler/config.json` and `scheduler/state.db`.
- Use the update script (`bash scripts/update.sh --restart`) instead of ad-hoc rebuilds.

## 6) Add a go/no-go review checklist

Before every config change or go-live:

- [ ] Strategy IDs and platforms are correct.
- [ ] Symbols/timeframes are exactly what you intended.
- [ ] Risk limits are non-zero and sane.
- [ ] Stop-loss logic is configured and tested.
- [ ] Notification channels receive alerts.
- [ ] `/health` and `/status` are clean.

---

## Optional hardening ideas

- Run two instances: one paper canary + one live.
- Add a daily script that snapshots `/status` and key balances.
- Add CI checks for config linting and a canned `--once` smoke run on staging.
