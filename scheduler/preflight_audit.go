package main

import (
	"fmt"
	"strings"
)

// PreflightIssue is a startup-readiness finding surfaced by --preflight.
type PreflightIssue struct {
	Severity string // "error" or "warn"
	Message  string
}

// BuildPreflightAudit reports common operator misconfigurations before live use.
// It intentionally focuses on high-signal checks and does not replace full
// config validation done by LoadConfig.
func BuildPreflightAudit(cfg *Config) []PreflightIssue {
	if cfg == nil {
		return []PreflightIssue{{Severity: "error", Message: "nil config"}}
	}
	issues := make([]PreflightIssue, 0)

	if cfg.PortfolioRisk == nil {
		issues = append(issues, PreflightIssue{Severity: "warn", Message: "portfolio_risk is not set; consider max_drawdown_pct and max_notional_usd rails"})
	} else {
		if cfg.PortfolioRisk.MaxDrawdownPct <= 0 {
			issues = append(issues, PreflightIssue{Severity: "error", Message: "portfolio_risk.max_drawdown_pct must be > 0 for safe live operation"})
		}
		if cfg.PortfolioRisk.WarnThresholdPct <= 0 {
			issues = append(issues, PreflightIssue{Severity: "warn", Message: "portfolio_risk.warn_threshold_pct is <= 0; drawdown warning alerts may never trigger"})
		}
	}

	for _, sc := range cfg.Strategies {
		id := strings.TrimSpace(sc.ID)
		if id == "" {
			id = "<unknown>"
		}
		if sc.MaxDrawdownPct <= 0 {
			issues = append(issues, PreflightIssue{Severity: "error", Message: fmt.Sprintf("strategy[%s].max_drawdown_pct must be > 0", id)})
		}
		if sc.Type == "perps" || sc.Type == "manual" {
			hasSL := (sc.StopLossPct != nil && *sc.StopLossPct > 0) ||
				(sc.StopLossMarginPct != nil && *sc.StopLossMarginPct > 0) ||
				(sc.StopLossATRMult != nil && *sc.StopLossATRMult > 0) ||
				(sc.StopLossATRRegime != nil) ||
				(sc.TrailingStopPct != nil && *sc.TrailingStopPct > 0) ||
				(sc.TrailingStopATRMult != nil && *sc.TrailingStopATRMult > 0) ||
				(sc.TrailingStopATRRegime != nil)
			if !hasSL {
				issues = append(issues, PreflightIssue{Severity: "warn", Message: fmt.Sprintf("strategy[%s] has no explicit stop-loss/trailing-stop configured", id)})
			}
		}
	}
	return issues
}
