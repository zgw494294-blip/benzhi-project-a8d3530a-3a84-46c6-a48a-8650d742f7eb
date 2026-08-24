package application

import (
	"testing"
	"time"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/repository"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/rules"
)

func TestFullWorkflowConcurrencyIdempotencyAndIndependence(t *testing.T) {
	store, err := repository.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store, rules.NewEngine())
	created, err := service.CreateCase(CreateCaseCommand{
		Meta: CommandMeta{Actor: "监测员甲", Role: domain.RoleMonitor, IdempotencyKey: "create-1"},
		Name: "南湾海草床修复", SiteCode: "SW-SG-01", HabitatType: "海草床",
		Baseline: []domain.IndicatorRange{{Indicator: "海草覆盖率", Minimum: 65, Maximum: 100, Unit: "%"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	retried, err := service.CreateCase(CreateCaseCommand{
		Meta: CommandMeta{Actor: "监测员甲", Role: domain.RoleMonitor, IdempotencyKey: "create-1"},
		Name: "不会覆盖", SiteCode: "OTHER", HabitatType: "其他",
		Baseline: []domain.IndicatorRange{{Indicator: "其他", Minimum: 0, Maximum: 1, Unit: "x"}},
	})
	if err != nil || !retried.Idempotent || retried.Case.ID != created.Case.ID {
		t.Fatalf("建档幂等重试不稳定: result=%#v err=%v", retried, err)
	}

	now := time.Now().UTC()
	monitored, err := service.SubmitMonitoring(SubmitMonitoringCommand{
		Meta:   CommandMeta{Actor: "监测员甲", Role: domain.RoleMonitor, ExpectedVersion: created.Case.Version, IdempotencyKey: "monitor-1"},
		CaseID: created.Case.ID, Indicator: "海草覆盖率", ObservedValue: 40, Unit: "%",
		EvidenceNote: "样带 T-03，水下影像 VID-100", CapturedBy: "监测员甲", CapturedAt: now,
		RemediationOwner: "整改员乙", RemediationDueAt: now.Add(48 * time.Hour),
	})
	if err != nil || monitored.Action == nil {
		t.Fatalf("偏差监测未生成整改: %#v err=%v", monitored, err)
	}
	if _, err := service.SubmitMonitoring(SubmitMonitoringCommand{
		Meta:   CommandMeta{Actor: "监测员甲", Role: domain.RoleMonitor, ExpectedVersion: created.Case.Version, IdempotencyKey: "monitor-stale"},
		CaseID: created.Case.ID, Indicator: "海草覆盖率", ObservedValue: 75, Unit: "%",
		EvidenceNote: "并发旧读数", CapturedBy: "监测员甲", CapturedAt: now,
	}); domain.ErrorCodeOf(err) != domain.CodeConflict {
		t.Fatalf("旧 expectedVersion 应冲突: %v", err)
	}
	if _, err := service.SubmitAcceptance(SubmitAcceptanceCommand{
		Meta:   CommandMeta{Actor: "复核员丙", Role: domain.RoleReviewer, ExpectedVersion: monitored.Case.Version, IdempotencyKey: "accept-too-early"},
		CaseID: created.Case.ID, Reviewer: "复核员丙", Decision: domain.DecisionAccepted, ReviewNote: "过早验收",
	}); domain.ErrorCodeOf(err) != domain.CodeInvalidState {
		t.Fatalf("未关闭整改不应验收: %v", err)
	}

	failedRetest, err := service.SubmitRetest(SubmitRetestCommand{
		Meta:   CommandMeta{Actor: "整改员乙", Role: domain.RoleRemediator, ExpectedVersion: monitored.Case.Version, IdempotencyKey: "retest-1"},
		CaseID: created.Case.ID, ActionID: monitored.Action.ID, Owner: "整改员乙",
		ObservedValue: 58, EvidenceNote: "首次补植后复测，尚未达标",
	})
	if err != nil || failedRetest.Action.Status != domain.RemediationOpen {
		t.Fatalf("未达标复测应保持整改开启: %#v err=%v", failedRetest, err)
	}
	passedRetest, err := service.SubmitRetest(SubmitRetestCommand{
		Meta:   CommandMeta{Actor: "整改员乙", Role: domain.RoleRemediator, ExpectedVersion: failedRetest.Case.Version, IdempotencyKey: "retest-2"},
		CaseID: created.Case.ID, ActionID: monitored.Action.ID, Owner: "整改员乙",
		ObservedValue: 74, EvidenceNote: "二次补植后回到基线范围",
	})
	if err != nil || passedRetest.Action.Status != domain.RemediationResolved {
		t.Fatalf("达标复测应关闭整改: %#v err=%v", passedRetest, err)
	}
	if _, err := service.SubmitAcceptance(SubmitAcceptanceCommand{
		Meta:   CommandMeta{Actor: "监测员甲", Role: domain.RoleReviewer, ExpectedVersion: passedRetest.Case.Version, IdempotencyKey: "accept-not-independent"},
		CaseID: created.Case.ID, Reviewer: "监测员甲", Decision: domain.DecisionAccepted, ReviewNote: "本人复核",
	}); domain.ErrorCodeOf(err) != domain.CodeForbidden {
		t.Fatalf("采集人不能独立复核: %v", err)
	}

	accepted, err := service.SubmitAcceptance(SubmitAcceptanceCommand{
		Meta:   CommandMeta{Actor: "复核员丙", Role: domain.RoleReviewer, ExpectedVersion: passedRetest.Case.Version, IdempotencyKey: "accept-1"},
		CaseID: created.Case.ID, Reviewer: "复核员丙", Decision: domain.DecisionAccepted,
		ReviewNote: "监测来源清楚，整改两次复测形成闭环，同意冻结",
	})
	if err != nil || accepted.Certificate == nil || accepted.Case.Status != domain.CaseAcceptanceFrozen {
		t.Fatalf("验收冻结失败: %#v err=%v", accepted, err)
	}
	retryAcceptance, err := service.SubmitAcceptance(SubmitAcceptanceCommand{
		Meta:   CommandMeta{Actor: "复核员丙", Role: domain.RoleReviewer, ExpectedVersion: passedRetest.Case.Version, IdempotencyKey: "accept-1"},
		CaseID: created.Case.ID, Reviewer: "复核员丙", Decision: domain.DecisionAccepted,
		ReviewNote: "重复请求",
	})
	if err != nil || !retryAcceptance.Idempotent || retryAcceptance.Certificate.CredentialCode != accepted.Certificate.CredentialCode {
		t.Fatalf("冻结命令幂等重试不稳定: %#v err=%v", retryAcceptance, err)
	}
}
