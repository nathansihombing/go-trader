package main

import (
	"strings"
	"testing"
)

func hasPreflightIssue(issues []PreflightIssue, severity, substr string) bool {
	for _, issue := range issues {
		if issue.Severity == severity && strings.Contains(issue.Message, substr) {
			return true
		}
	}
	return false
}

func TestBuildPreflightAuditFlagsRiskRailErrors(t *testing.T) {
	cfg := &Config{
		PortfolioRisk: &PortfolioRiskConfig{MaxDrawdownPct: 0},
		Strategies: []StrategyConfig{{
			ID:             "bad-dd",
			Type:           "spot",
			MaxDrawdownPct: 0,
		}},
	}

	issues := BuildPreflightAudit(cfg)
	if !hasPreflightIssue(issues, "error", "portfolio_risk.max_drawdown_pct") {
		t.Fatalf("missing portfolio drawdown error: %#v", issues)
	}
	if !hasPreflightIssue(issues, "error", "strategy[bad-dd].max_drawdown_pct") {
		t.Fatalf("missing strategy drawdown error: %#v", issues)
	}
}

func TestBuildPreflightAuditWarnsOnPerpsWithoutExplicitStop(t *testing.T) {
	cfg := &Config{
		PortfolioRisk: &PortfolioRiskConfig{MaxDrawdownPct: 25, WarnThresholdPct: 60},
		Strategies: []StrategyConfig{{
			ID:             "hl-no-sl",
			Type:           "perps",
			MaxDrawdownPct: 5,
		}},
	}

	issues := BuildPreflightAudit(cfg)
	if !hasPreflightIssue(issues, "warn", "no explicit stop-loss/trailing-stop") {
		t.Fatalf("missing stop-loss warning: %#v", issues)
	}
}

func TestBuildPreflightAuditAcceptsPerpsWithStop(t *testing.T) {
	stop := 1.5
	cfg := &Config{
		PortfolioRisk: &PortfolioRiskConfig{MaxDrawdownPct: 25, WarnThresholdPct: 60},
		Strategies: []StrategyConfig{{
			ID:              "hl-with-sl",
			Type:            "perps",
			MaxDrawdownPct:  5,
			StopLossATRMult: &stop,
		}},
	}

	issues := BuildPreflightAudit(cfg)
	if len(issues) != 0 {
		t.Fatalf("expected no preflight issues, got %#v", issues)
	}
}
