package stalecasedetailcache_test

import (
	"testing"
	"time"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/application"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/repository"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/rules"
)

func TestCaseDetailCacheRefreshesAfterWrite(t *testing.T) {
	store, err := repository.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := application.NewService(store, rules.NewEngine())
	created, err := service.CreateCase(application.CreateCaseCommand{
		Meta: application.CommandMeta{
			Actor:          "监测员甲",
			Role:           domain.RoleMonitor,
			IdempotencyKey: "create-cache-case",
		},
		Name:        "缓存一致性验证案件",
		SiteCode:    "CACHE-01",
		HabitatType: "海草床",
		Baseline: []domain.IndicatorRange{{
			Indicator: "海草覆盖率",
			Minimum:   65,
			Maximum:   100,
			Unit:      "%",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	primed, err := service.GetCase(created.Case.ID)
	if err != nil {
		t.Fatal(err)
	}
	if primed.Case.Version != 1 || len(primed.Timeline) != 1 {
		t.Fatalf("缓存预热前的案件详情异常: version=%d timeline=%d", primed.Case.Version, len(primed.Timeline))
	}

	recorded, err := service.SubmitMonitoring(application.SubmitMonitoringCommand{
		Meta: application.CommandMeta{
			Actor:           "监测员甲",
			Role:            domain.RoleMonitor,
			ExpectedVersion: primed.Case.Version,
			IdempotencyKey:  "monitor-after-cache",
		},
		CaseID:        created.Case.ID,
		Indicator:     "海草覆盖率",
		ObservedValue: 82,
		Unit:          "%",
		EvidenceNote:  "样带 CACHE-T1 的现场影像",
		CapturedBy:    "监测员甲",
		CapturedAt:    time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if recorded.Case.Version != 2 {
		t.Fatalf("监测写入未提交: version=%d", recorded.Case.Version)
	}

	refreshed, err := service.GetCase(created.Case.ID)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.Case.Version != recorded.Case.Version ||
		len(refreshed.Case.Monitoring) != 1 ||
		len(refreshed.Timeline) != 2 {
		t.Fatalf(
			"写入后案件详情仍来自旧缓存: version=%d monitoring=%d timeline=%d",
			refreshed.Case.Version,
			len(refreshed.Case.Monitoring),
			len(refreshed.Timeline),
		)
	}
}
