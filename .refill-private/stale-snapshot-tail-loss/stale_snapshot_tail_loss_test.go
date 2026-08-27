package stale_snapshot_tail_loss_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/repository"
)

func TestRestartReplaysLedgerTailAfterStaleSnapshot(t *testing.T) {
	dataDir := t.TempDir()
	store, err := repository.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, time.August, 25, 8, 0, 0, 0, time.UTC)
	item, err := domain.NewRestorationCase(
		"case-snapshot-tail",
		"滩涂植被恢复",
		"TF-VEG-01",
		"滩涂",
		[]domain.IndicatorRange{{Indicator: "植被覆盖率", Minimum: 60, Maximum: 100, Unit: "%"}},
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	created, err := domain.NewEvent(
		"evt-snapshot-created",
		item.ID,
		domain.EventCaseCreated,
		"监测员甲",
		"snapshot-create",
		1,
		now,
		domain.CaseCreatedData{Case: *item},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Append(item.ID, 0, []domain.Event{created}); err != nil {
		t.Fatal(err)
	}

	snapshotPath := filepath.Join(dataDir, "snapshot.json")
	staleSnapshot, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatal(err)
	}

	rangeValue, err := item.BaselineFor("植被覆盖率")
	if err != nil {
		t.Fatal(err)
	}
	record := domain.MonitoringRecord{
		ID: "mon-snapshot-tail", CaseID: item.ID, Indicator: rangeValue.Indicator,
		ExpectedRange: rangeValue, ObservedValue: 78, Unit: "%",
		EvidenceNote: "样方 A-07 影像证据", CapturedBy: "监测员甲", CapturedAt: now.Add(time.Hour),
		Status: domain.MonitoringPass,
	}
	monitored, err := domain.NewEvent(
		"evt-snapshot-monitoring",
		item.ID,
		domain.EventMonitoringRecorded,
		"监测员甲",
		"snapshot-monitoring",
		2,
		now.Add(time.Hour),
		domain.MonitoringRecordedData{Record: record},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Append(item.ID, 1, []domain.Event{monitored}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(snapshotPath, staleSnapshot, 0o640); err != nil {
		t.Fatal(err)
	}

	recovered, err := repository.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := recovered.Get(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != 2 || len(got.Monitoring) != 1 {
		t.Fatalf("重启丢失账本尾部事件: version=%d monitoring=%d", got.Version, len(got.Monitoring))
	}
	sequence, _ := recovered.Integrity()
	if sequence != 2 {
		t.Fatalf("恢复投影序号 = %d, want 2", sequence)
	}
}
