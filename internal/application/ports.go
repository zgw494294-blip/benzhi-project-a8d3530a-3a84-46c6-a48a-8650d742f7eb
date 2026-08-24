package application

import "benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"

type Store interface {
	Append(caseID string, expectedVersion int64, events []domain.Event) (*domain.RestorationCase, error)
	Get(caseID string) (*domain.RestorationCase, error)
	List() []*domain.RestorationCase
	Events(caseID string) ([]domain.Event, error)
	Certificate(code string) (*domain.AcceptanceCertificate, error)
	IdempotentCase(key string) (*domain.RestorationCase, bool)
	Integrity() (int64, string)
}
