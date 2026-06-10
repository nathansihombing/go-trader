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

func TestPreflightExitCodeStrictModeFailsWarnings(t *testing.T) {
	issues := []PreflightIssue{{Severity: "warn", Message: "warning only"}}
	if got := PreflightExitCode(issues, false); got != preflightExitOK {
		t.Fatalf("non-strict warning exit = %d, want %d", got, preflightExitOK)
	}
	if got := PreflightExitCode(issues, true); got != preflightExitError {
		t.Fatalf("strict warning exit = %d, want %d", got, preflightExitError)
	}
}

func TestPreflightExitCodeErrorsAlwaysFail(t *testing.T) {
	issues := []PreflightIssue{{Severity: "error", Message: "bad"}}
	if got := PreflightExitCode(issues, false); got != preflightExitError {
		t.Fatalf("non-strict error exit = %d, want %d", got, preflightExitError)
	}
	if got := PreflightExitCode(issues, true); got != preflightExitError {
		t.Fatalf("strict error exit = %d, want %d", got, preflightExitError)
	}
}

func TestPreflightExitCodeCleanPasses(t *testing.T) {
	if got := PreflightExitCode(nil, true); got != preflightExitOK {
		t.Fatalf("clean strict exit = %d, want %d", got, preflightExitOK)
	}
}

func TestBuildPreflightReportStatusAndJSONShape(t *testing.T) {
	issues := []PreflightIssue{{Severity: "warn", Message: "warning only"}}
	report := BuildPreflightReport(issues, true)
	if report.Status != "warn" {
		t.Fatalf("status = %q, want warn", report.Status)
	}
	if !report.Strict {
		t.Fatal("expected strict report")
	}
	if report.ExitCode != preflightExitError {
		t.Fatalf("exit code = %d, want %d", report.ExitCode, preflightExitError)
	}
	if len(report.Issues) != 1 || report.Issues[0].Severity != "warn" || report.Issues[0].Message == "" {
		t.Fatalf("unexpected issues in report: %#v", report.Issues)
	}
}

func TestBuildPreflightReportCleanUsesEmptyIssueSlice(t *testing.T) {
	report := BuildPreflightReport(nil, false)
	if report.Status != "ok" {
		t.Fatalf("status = %q, want ok", report.Status)
	}
	if report.ExitCode != preflightExitOK {
		t.Fatalf("exit code = %d, want %d", report.ExitCode, preflightExitOK)
	}
	if report.Issues == nil || len(report.Issues) != 0 {
		t.Fatalf("expected non-nil empty issue slice, got %#v", report.Issues)
	}
}

func TestPreflightStatusPrefersErrors(t *testing.T) {
	issues := []PreflightIssue{
		{Severity: "warn", Message: "warning"},
		{Severity: "error", Message: "error"},
	}
	if got := PreflightStatus(issues); got != "error" {
		t.Fatalf("status = %q, want error", got)
	}
}
