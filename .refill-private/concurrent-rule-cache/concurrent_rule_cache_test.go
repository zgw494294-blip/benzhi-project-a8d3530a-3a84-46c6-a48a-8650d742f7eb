package concurrent_rule_cache_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/application"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/rules"
)

type independentAppendStore struct {
	base *domain.RestorationCase
}

func (s *independentAppendStore) Append(_ string, _ int64, events []domain.Event) (*domain.RestorationCase, error) {
	item := s.base.Clone()
	for _, event := range events {
		if err := domain.ApplyEvent(item, event); err != nil {
			return nil, err
		}
	}
	return item, nil
}

func (s *independentAppendStore) Get(string) (*domain.RestorationCase, error) {
	return s.base.Clone(), nil
}

func (s *independentAppendStore) List() []*domain.RestorationCase { return nil }

func (s *independentAppendStore) Events(string) ([]domain.Event, error) { return nil, nil }

func (s *independentAppendStore) Certificate(string) (*domain.AcceptanceCertificate, error) {
	return nil, domain.NewError(domain.CodeNotFound, "凭据不存在")
}

func (s *independentAppendStore) IdempotentCase(string) (*domain.RestorationCase, bool) {
	return nil, false
}

func (s *independentAppendStore) Integrity() (int64, string) { return 0, "" }

func TestConcurrentMonitoringAssessmentsSynchronizeRuleCache(t *testing.T) {
	now := time.Now().UTC()
	item, err := domain.NewRestorationCase(
		"case-concurrent-rules",
		"并发监测规则案件",
		"RULE-RACE-01",
		"海草床",
		[]domain.IndicatorRange{{Indicator: "覆盖率", Minimum: 0, Maximum: 100000, Unit: "%"}},
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	service := application.NewService(&independentAppendStore{base: item}, rules.NewEngine())

	const workers = 12
	const iterations = 64
	start := make(chan struct{})
	errors := make(chan error, workers)
	var ready sync.WaitGroup
	var done sync.WaitGroup
	ready.Add(workers)
	done.Add(workers)
	for worker := 0; worker < workers; worker++ {
		go func(worker int) {
			defer done.Done()
			ready.Done()
			<-start
			for iteration := 0; iteration < iterations; iteration++ {
				sequence := worker*iterations + iteration
				_, submitErr := service.SubmitMonitoring(application.SubmitMonitoringCommand{
					Meta: application.CommandMeta{
						ExpectedVersion: 0,
						IdempotencyKey:  fmt.Sprintf("rule-cache-%d", sequence),
						Actor:           fmt.Sprintf("监测员-%d", worker),
						Role:            domain.RoleMonitor,
					},
					CaseID:        item.ID,
					Indicator:     "覆盖率",
					ObservedValue: float64(sequence),
					Unit:          "%",
					EvidenceNote:  "并发采样证据",
					CapturedAt:    now,
				})
				if submitErr != nil {
					errors <- submitErr
					return
				}
			}
		}(worker)
	}
	ready.Wait()
	close(start)
	done.Wait()
	close(errors)
	for submitErr := range errors {
		t.Fatalf("并发监测提交失败: %v", submitErr)
	}
}
