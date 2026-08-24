package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
)

func TestStoreRecoversProjectionAndRejectsStaleVersion(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	item, err := domain.NewRestorationCase("case-1", "北湾盐沼", "BW-SM-01", "盐沼", []domain.IndicatorRange{
		{Indicator: "植被覆盖率", Minimum: 60, Maximum: 100, Unit: "%"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	event, err := domain.NewEvent("evt-1", item.ID, domain.EventCaseCreated, "监测员", "key-1", 1, now, domain.CaseCreatedData{Case: *item})
	if err != nil {
		t.Fatal(err)
	}
	stored, err := store.Append(item.ID, 0, []domain.Event{event})
	if err != nil {
		t.Fatal(err)
	}
	if stored.Version != 1 {
		t.Fatalf("版本 = %d, want 1", stored.Version)
	}
	if _, err := store.Append(item.ID, 0, []domain.Event{event}); domain.ErrorCodeOf(err) != domain.CodeConflict {
		t.Fatalf("旧版本追加应冲突: %v", err)
	}
	recovered, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := recovered.Get(item.ID)
	if err != nil || got.BaselineVersion != item.BaselineVersion {
		t.Fatalf("重放未恢复案件: got=%#v err=%v", got, err)
	}
	sequence, hash := recovered.Integrity()
	if sequence != 1 || len(hash) != 64 {
		t.Fatalf("账本完整性投影异常: sequence=%d hash=%q", sequence, hash)
	}
}

func TestOpenDetectsLedgerTampering(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	item, _ := domain.NewRestorationCase("case-tamper", "东湾红树林", "DW-MG-01", "红树林", []domain.IndicatorRange{
		{Indicator: "覆盖率", Minimum: 70, Maximum: 100, Unit: "%"},
	}, now)
	event, _ := domain.NewEvent("evt-tamper", item.ID, domain.EventCaseCreated, "监测员", "key-tamper", 1, now, domain.CaseCreatedData{Case: *item})
	if _, err := store.Append(item.ID, 0, []domain.Event{event}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "events.jsonl")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(string(content), "DW-MG-01", "DW-MG-02", 1)
	if err := os.WriteFile(path, []byte(tampered), 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(dir); domain.ErrorCodeOf(err) != domain.CodeIntegrity {
		t.Fatalf("篡改账本应触发完整性错误: %v", err)
	}
}
