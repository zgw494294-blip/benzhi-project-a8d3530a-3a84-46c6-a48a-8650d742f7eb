package repository

import (
	"strings"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
)

func (s *LocalStore) Get(caseID string) (*domain.RestorationCase, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item := s.projection.Cases[caseID]
	if item == nil {
		return nil, domain.NewError(domain.CodeNotFound, "案件 %s 不存在", caseID)
	}
	return item.Clone(), nil
}

func (s *LocalStore) List() []*domain.RestorationCase {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.projection.caseList()
}

func (s *LocalStore) Events(caseID string) ([]domain.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.projection.Cases[caseID] == nil {
		return nil, domain.NewError(domain.CodeNotFound, "案件 %s 不存在", caseID)
	}
	return s.projection.Events[caseID], nil
}

func (s *LocalStore) Certificate(code string) (*domain.AcceptanceCertificate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	caseID := s.projection.Credentials[strings.ToUpper(strings.TrimSpace(code))]
	if caseID == "" {
		return nil, domain.NewError(domain.CodeNotFound, "凭据 %s 不存在", code)
	}
	certificate := *s.projection.Cases[caseID].Certificate
	return &certificate, nil
}

func (s *LocalStore) IdempotentCase(key string) (*domain.RestorationCase, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	caseID, ok := s.projection.Idempotency[key]
	if !ok {
		return nil, false
	}
	return s.projection.Cases[caseID].Clone(), true
}

func (s *LocalStore) Integrity() (int64, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.projection.LastSequence, s.projection.LastHash
}
