package application

import (
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
)

type CaseDetail struct {
	Case     *domain.RestorationCase `json:"case"`
	Timeline []domain.Event          `json:"timeline"`
}

func (s *Service) GetCase(caseID string) (CaseDetail, error) {
	s.detailMu.RLock()
	cached, ok := s.detailCache[caseID]
	s.detailMu.RUnlock()
	if ok {
		return cloneCaseDetail(cached), nil
	}
	item, err := s.store.Get(caseID)
	if err != nil {
		return CaseDetail{}, err
	}
	events, err := s.store.Events(caseID)
	if err != nil {
		return CaseDetail{}, err
	}
	detail := CaseDetail{Case: item, Timeline: events}
	s.detailMu.Lock()
	s.detailCache[caseID] = detail
	s.detailMu.Unlock()
	return cloneCaseDetail(detail), nil
}

func cloneCaseDetail(detail CaseDetail) CaseDetail {
	clone := CaseDetail{Case: detail.Case.Clone()}
	clone.Timeline = make([]domain.Event, len(detail.Timeline))
	for index, event := range detail.Timeline {
		clone.Timeline[index] = event
		clone.Timeline[index].Data = append([]byte(nil), event.Data...)
	}
	return clone
}

func (s *Service) ListCases() []*domain.RestorationCase { return s.store.List() }

func (s *Service) GetCertificate(code string) (*domain.AcceptanceCertificate, error) {
	return s.store.Certificate(code)
}

type SystemIntegrity struct {
	Sequence int64  `json:"sequence"`
	LastHash string `json:"lastHash"`
}

func (s *Service) Integrity() SystemIntegrity {
	sequence, hash := s.store.Integrity()
	return SystemIntegrity{Sequence: sequence, LastHash: hash}
}
