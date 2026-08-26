package splitacceptancecommit_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/application"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/repository"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/rules"
)

var errFreezeUnavailable = errors.New("验收冻结存储暂时不可用")

type freezeFailStore struct {
	*repository.LocalStore
}

func (s *freezeFailStore) Append(caseID string, expectedVersion int64, events []domain.Event) (*domain.RestorationCase, error) {
	for _, event := range events {
		if event.Type == domain.EventAcceptanceFrozen {
			return nil, errFreezeUnavailable
		}
	}
	return s.LocalStore.Append(caseID, expectedVersion, events)
}

func TestAcceptanceFailureDoesNotPersistPartialReview(t *testing.T) {
	store, err := repository.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := application.NewService(&freezeFailStore{LocalStore: store}, rules.NewEngine())
	created, err := service.CreateCase(application.CreateCaseCommand{
		Meta: application.CommandMeta{Actor: "监测员甲", Role: domain.RoleMonitor, IdempotencyKey: "atomic-create"},
		Name: "东湾红树林恢复", SiteCode: "DW-MG-ATOMIC", HabitatType: "红树林",
		Baseline: []domain.IndicatorRange{{Indicator: "植被覆盖率", Minimum: 70, Maximum: 100, Unit: "%"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	monitored, err := service.SubmitMonitoring(application.SubmitMonitoringCommand{
		Meta: application.CommandMeta{
			Actor: "监测员甲", Role: domain.RoleMonitor,
			ExpectedVersion: created.Case.Version, IdempotencyKey: "atomic-monitor",
		},
		CaseID: created.Case.ID, Indicator: "植被覆盖率", ObservedValue: 82, Unit: "%",
		EvidenceNote: "样方 A-07 影像完整", CapturedBy: "监测员甲", CapturedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.SubmitAcceptance(application.SubmitAcceptanceCommand{
		Meta: application.CommandMeta{
			Actor: "复核员乙", Role: domain.RoleReviewer,
			ExpectedVersion: monitored.Case.Version, IdempotencyKey: "atomic-accept",
		},
		CaseID: created.Case.ID, Reviewer: "复核员乙", Decision: domain.DecisionAccepted,
		ReviewNote: "证据完整，同意冻结",
	})
	if err == nil || !strings.Contains(err.Error(), errFreezeUnavailable.Error()) {
		t.Fatalf("冻结故障应返回原始存储错误: %v", err)
	}

	after, err := store.Get(created.Case.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Version != monitored.Case.Version || len(after.Reviews) != 0 || after.Certificate != nil {
		t.Fatalf("冻结失败后验收命令发生部分提交: version=%d reviews=%d certificate=%v", after.Version, len(after.Reviews), after.Certificate)
	}
}
