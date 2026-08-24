package repository

import (
	"sort"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
)

type projection struct {
	Cases        map[string]*domain.RestorationCase `json:"cases"`
	Events       map[string][]domain.Event          `json:"events"`
	Credentials  map[string]string                  `json:"credentials"`
	Idempotency  map[string]string                  `json:"idempotency"`
	LastSequence int64                              `json:"lastSequence"`
	LastHash     string                             `json:"lastHash"`
}

func newProjection() *projection {
	return &projection{
		Cases: make(map[string]*domain.RestorationCase), Events: make(map[string][]domain.Event),
		Credentials: make(map[string]string), Idempotency: make(map[string]string),
	}
}

func (p *projection) apply(entry LedgerEntry) error {
	event := entry.Event
	current := p.Cases[event.CaseID]
	if current == nil {
		if event.Type != domain.EventCaseCreated {
			return domain.NewError(domain.CodeIntegrity, "案件 %s 的首个事件不是建档事件", event.CaseID)
		}
		current = &domain.RestorationCase{}
		p.Cases[event.CaseID] = current
	}
	if err := domain.ApplyEvent(current, event); err != nil {
		return err
	}
	p.Events[event.CaseID] = append(p.Events[event.CaseID], event)
	if event.IdempotencyKey != "" {
		p.Idempotency[event.IdempotencyKey] = event.CaseID
	}
	if current.Certificate != nil {
		p.Credentials[current.Certificate.CredentialCode] = current.ID
	}
	p.LastSequence = entry.Sequence
	p.LastHash = entry.Hash
	return nil
}

func replay(entries []LedgerEntry) (*projection, error) {
	p := newProjection()
	for _, entry := range entries {
		if err := p.apply(entry); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func (p *projection) caseList() []*domain.RestorationCase {
	items := make([]*domain.RestorationCase, 0, len(p.Cases))
	for _, item := range p.Cases {
		items = append(items, item.Clone())
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return items
}
