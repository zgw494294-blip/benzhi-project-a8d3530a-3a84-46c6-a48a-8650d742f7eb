package resource_save_state_pollution_test

import (
	"os"
	"path/filepath"
	"testing"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/application"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/repository"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/rules"
)

func TestFailedLedgerWriteDoesNotPolluteStore(t *testing.T) {
	dataDir := t.TempDir()
	store, err := repository.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(dataDir, "events.jsonl")
	if err := os.Remove(ledgerPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(ledgerPath, 0o750); err != nil {
		t.Fatal(err)
	}

	service := application.NewService(store, rules.NewEngine())
	_, err = service.CreateCase(application.CreateCaseCommand{
		Meta: application.CommandMeta{
			Actor:          "监测员甲",
			Role:           domain.RoleMonitor,
			IdempotencyKey: "failed-create",
		},
		Name:        "资源失效测试案件",
		SiteCode:    "FAIL-01",
		HabitatType: "盐沼",
		Baseline: []domain.IndicatorRange{
			{Indicator: "植被覆盖率", Minimum: 60, Maximum: 100, Unit: "%"},
		},
	})
	if err == nil {
		t.Fatal("账本路径失效时建档应返回错误")
	}
	if cases := store.List(); len(cases) != 0 {
		t.Fatalf("账本追加失败后内存投影包含 %d 个未持久化案件", len(cases))
	}
}
