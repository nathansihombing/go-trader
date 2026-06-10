package main

import (
	"fmt"
	"os"
	"strings"
)

// PreflightIssue is a startup-readiness finding surfaced by --preflight.
type PreflightIssue struct {
	Severity string `json:"severity"` // "error" or "warn"
	Message  string `json:"message"`
}

// PreflightReport is the machine-readable summary emitted by --preflight-json.
type PreflightReport struct {
	Status   string           `json:"status"`
	Strict   bool             `json:"strict"`
	ExitCode int              `json:"exit_code"`
	Issues   []PreflightIssue `json:"issues"`
}

const (
	preflightExitOK    = 0
	preflightExitError = 2
)

// PreflightExitCode returns the process exit code for audit findings. In
// strict mode warnings are treated as failures so CI/deploy scripts can enforce
// a fully clean preflight before going live.
func PreflightExitCode(issues []PreflightIssue, strict bool) int {
	for _, it := range issues {
		if it.Severity == "error" || (strict && it.Severity == "warn") {
			return preflightExitError
		}
	}
	return preflightExitOK
}

func PreflightStatus(issues []PreflightIssue) string {
	status := "ok"
	for _, it := range issues {
		switch it.Severity {
		case "error":
			return "error"
		case "warn":
			if status == "ok" {
				status = "warn"
			}
		}
	}
	return status
}

func BuildPreflightReport(issues []PreflightIssue, strict bool) PreflightReport {
	if issues == nil {
		issues = []PreflightIssue{}
	}
	return PreflightReport{
		Status:   PreflightStatus(issues),
		Strict:   strict,
		ExitCode: PreflightExitCode(issues, strict),
		Issues:   issues,
	}
}

func argsMode(args []string) string {
	for i, arg := range args {
		if strings.HasPrefix(arg, "--mode=") {
			return strings.TrimSpace(strings.TrimPrefix(arg, "--mode="))
		}
		if arg == "--mode" && i+1 < len(args) {
			return strings.TrimSpace(args[i+1])
		}
	}
	return ""
}

func strategyIsLive(sc StrategyConfig) bool {
	if sc.Type == "manual" {
		return true
	}
	return strings.EqualFold(argsMode(sc.Args), "live")
}

func missingEnvVars(names ...string) []string {
	missing := make([]string, 0, len(names))
	for _, name := range names {
		if strings.TrimSpace(os.Getenv(name)) == "" {
			missing = append(missing, name)
		}
	}
	return missing
}

func liveCredentialEnvVars(sc StrategyConfig) []string {
	if !strategyIsLive(sc) {
		return nil
	}
	switch sc.Platform {
	case "hyperliquid":
		return []string{"HYPERLIQUID_SECRET_KEY"}
	case "topstep":
		return []string{"TOPSTEP_API_KEY", "TOPSTEP_API_SECRET", "TOPSTEP_ACCOUNT_ID"}
	case "robinhood":
		return []string{"ROBINHOOD_USERNAME", "ROBINHOOD_PASSWORD", "ROBINHOOD_TOTP_SECRET"}
	case "okx":
		return []string{"OKX_API_KEY", "OKX_API_SECRET", "OKX_PASSPHRASE"}
	}
	return nil
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
		if envVars := liveCredentialEnvVars(sc); len(envVars) > 0 {
			if missing := missingEnvVars(envVars...); len(missing) > 0 {
				issues = append(issues, PreflightIssue{Severity: "error", Message: fmt.Sprintf("strategy[%s] live %s requires missing env vars: %s", id, sc.Platform, strings.Join(missing, ", "))})
			}
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
