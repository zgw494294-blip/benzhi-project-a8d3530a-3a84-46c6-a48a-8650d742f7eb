package timeline_event_alias

import (
	"bytes"
	"testing"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/application"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/repository"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/rules"
)

func TestTimelineReadDoesNotAliasProjectionEvents(t *testing.T) {
	store, err := repository.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := application.NewService(store, rules.NewEngine())
	created, err := service.CreateCase(application.CreateCaseCommand{
		Meta: application.CommandMeta{Actor: "监测员甲", Role: domain.RoleMonitor, IdempotencyKey: "create-timeline-alias"},
		Name: "潮间带植被恢复", SiteCode: "TW-01", HabitatType: "盐沼",
		Baseline: []domain.IndicatorRange{{Indicator: "覆盖率", Minimum: 60, Maximum: 100, Unit: "%"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	timeline, err := store.Events(created.Case.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(timeline) != 1 {
		t.Fatalf("初始时间线长度 = %d, want 1", len(timeline))
	}
	originalID, originalType := timeline[0].ID, timeline[0].Type
	originalData := append([]byte(nil), timeline[0].Data...)
	timeline[0] = domain.Event{ID: "forged-event", CaseID: created.Case.ID, Type: "forged.type"}

	fresh, err := store.Events(created.Case.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fresh[0].ID != originalID || fresh[0].Type != originalType {
		t.Fatalf("时间线读取暴露了仓储投影别名: got id=%q type=%q, want id=%q type=%q", fresh[0].ID, fresh[0].Type, originalID, originalType)
	}
	fresh[0].Data[0] ^= 0xff
	latest, err := store.Events(created.Case.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(latest[0].Data, originalData) {
		t.Fatal("时间线事件的 Data 字节仍与仓储投影共享底层数组")
	}
}
