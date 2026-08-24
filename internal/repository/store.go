package repository

import (
	"fmt"
	"path/filepath"
	"sync"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
)

type LocalStore struct {
	mu           sync.RWMutex
	ledger       *Ledger
	projection   *projection
	snapshotPath string
}

func Open(dataDir string) (*LocalStore, error) {
	ledgerPath := filepath.Join(dataDir, "events.jsonl")
	snapshotPath := filepath.Join(dataDir, "snapshot.json")
	ledger, entries, err := openLedger(ledgerPath)
	if err != nil {
		return nil, err
	}
	p, err := replay(entries)
	if err != nil {
		return nil, fmt.Errorf("重放事件账本: %w", err)
	}
	if existing, loadErr := loadSnapshot(snapshotPath); loadErr == nil && existing != nil {
		if existing.Projection.LastSequence > p.LastSequence {
			return nil, domain.NewError(domain.CodeIntegrity, "快照序号超前于事件账本")
		}
	}
	store := &LocalStore{ledger: ledger, projection: p, snapshotPath: snapshotPath}
	if err := saveSnapshot(snapshotPath, p); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *LocalStore) Append(caseID string, expectedVersion int64, events []domain.Event) (*domain.RestorationCase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.projection.Cases[caseID]
	actualVersion := int64(0)
	if current != nil {
		actualVersion = current.Version
	}
	if actualVersion != expectedVersion {
		return nil, domain.NewError(domain.CodeConflict, "案件版本冲突：期望 %d，实际 %d", expectedVersion, actualVersion)
	}
	if current == nil && (len(events) == 0 || events[0].Type != domain.EventCaseCreated) {
		return nil, domain.NewError(domain.CodeNotFound, "案件 %s 不存在", caseID)
	}
	working := &domain.RestorationCase{}
	if current != nil {
		working = current.Clone()
	}
	for index, event := range events {
		if event.CaseID != caseID || event.Version != expectedVersion+int64(index)+1 {
			return nil, domain.NewError(domain.CodeIntegrity, "待提交事件的案件或版本无效")
		}
		if err := domain.ApplyEvent(working, event); err != nil {
			return nil, err
		}
	}
	entries, err := s.ledger.appendBatch(events)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if err := s.projection.apply(entry); err != nil {
			return nil, err
		}
	}
	if err := saveSnapshot(s.snapshotPath, s.projection); err != nil {
		return nil, err
	}
	return s.projection.Cases[caseID].Clone(), nil
}
