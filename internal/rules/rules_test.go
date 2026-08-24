package rules

import (
	"testing"
	"time"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
)

func TestAssessAndRetest(t *testing.T) {
	engine := NewEngine()
	expected := domain.IndicatorRange{Indicator: "植被覆盖率", Minimum: 70, Maximum: 100, Unit: "%"}
	pass := engine.Assess("红树林", expected, 82)
	if !pass.Passed || pass.RuleHit.RiskLevel != "none" {
		t.Fatalf("范围内读数应通过: %#v", pass)
	}
	deviation := engine.Assess("红树林", expected, 50)
	if deviation.Passed || deviation.RuleHit.RiskLevel == "none" {
		t.Fatalf("范围外读数应形成风险: %#v", deviation)
	}
	original := domain.MonitoringRecord{
		Indicator: "植被覆盖率", ExpectedRange: expected, ObservedValue: 50,
		Unit: "%", EvidenceNote: "样方证据", CapturedBy: "监测员", CapturedAt: time.Now(),
	}
	stillOpen := engine.AssessRetest("红树林", original, 65)
	if stillOpen.Closed || stillOpen.Improvement <= 0 {
		t.Fatalf("虽改善但未达标的复测不能关闭: %#v", stillOpen)
	}
	closed := engine.AssessRetest("红树林", original, 76)
	if !closed.Closed || closed.RuleHit.Rule != "retest.closed" {
		t.Fatalf("回到基线范围的复测应关闭: %#v", closed)
	}
}

func TestHabitatSensitivityRaisesRisk(t *testing.T) {
	engine := NewEngine()
	expected := domain.IndicatorRange{Indicator: "盐度", Minimum: 20, Maximum: 30, Unit: "ppt"}
	general := engine.Assess("其他", expected, 32.2)
	coral := engine.Assess("珊瑚礁", expected, 32.2)
	if general.RuleHit.RiskLevel != "medium" || coral.RuleHit.RiskLevel != "high" {
		t.Fatalf("敏感栖息地应提高同等偏差风险: general=%s coral=%s", general.RuleHit.RiskLevel, coral.RuleHit.RiskLevel)
	}
}
