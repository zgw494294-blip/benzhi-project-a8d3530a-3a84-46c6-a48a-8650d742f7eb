package application

import "benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"

type CaseDetail struct {
	Case     *domain.RestorationCase `json:"case"`
	Timeline []domain.Event          `json:"timeline"`
}

func (s *Service) GetCase(caseID string) (CaseDetail, error) {
	item, err := s.store.Get(caseID)
	if err != nil {
		return CaseDetail{}, err
	}
	events, err := s.store.Events(caseID)
	if err != nil {
		return CaseDetail{}, err
	}
	return CaseDetail{Case: item, Timeline: events}, nil
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
