package shared_ledger_writers_test

import (
	"sync"
	"testing"
	"time"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/repository"
)

func TestConcurrentStoresPreserveLedgerChain(t *testing.T) {
	directory := t.TempDir()
	first, err := repository.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	second, err := repository.Open(directory)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	makeEvent := func(caseID, eventID, key string) domain.Event {
		item, createErr := domain.NewRestorationCase(caseID, "共享账本案件", caseID, "盐沼", []domain.IndicatorRange{
			{Indicator: "覆盖率", Minimum: 60, Maximum: 100, Unit: "%"},
		}, now)
		if createErr != nil {
			t.Fatal(createErr)
		}
		event, eventErr := domain.NewEvent(eventID, caseID, domain.EventCaseCreated, "monitor-a", key, 1, now, domain.CaseCreatedData{Case: *item})
		if eventErr != nil {
			t.Fatal(eventErr)
		}
		return event
	}

	type appendCall struct {
		store  *repository.LocalStore
		caseID string
		event  domain.Event
	}
	calls := []appendCall{
		{store: first, caseID: "case-first", event: makeEvent("case-first", "event-first", "key-first")},
		{store: second, caseID: "case-second", event: makeEvent("case-second", "event-second", "key-second")},
	}
	ready := make(chan struct{}, len(calls))
	start := make(chan struct{})
	errors := make(chan error, len(calls))
	var workers sync.WaitGroup
	workers.Add(len(calls))
	for _, call := range calls {
		go func(call appendCall) {
			defer workers.Done()
			ready <- struct{}{}
			<-start
			_, appendErr := call.store.Append(call.caseID, 0, []domain.Event{call.event})
			errors <- appendErr
		}(call)
	}
	for range calls {
		<-ready
	}
	close(start)
	workers.Wait()
	close(errors)
	for appendErr := range errors {
		if appendErr != nil {
			t.Fatalf("共享账本的并发追加不应丢失任一已确认写入: %v", appendErr)
		}
	}

	recovered, err := repository.Open(directory)
	if err != nil {
		t.Fatalf("两个 store 确认写入后账本无法重启校验: %v", err)
	}
	if cases := recovered.List(); len(cases) != 2 {
		t.Fatalf("重启后案件数 = %d，want 2", len(cases))
	}
}
