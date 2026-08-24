package application

import (
	"strings"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
)

type CreateCaseCommand struct {
	Meta        CommandMeta             `json:"meta"`
	Name        string                  `json:"name"`
	SiteCode    string                  `json:"siteCode"`
	HabitatType string                  `json:"habitatType"`
	Baseline    []domain.IndicatorRange `json:"baseline"`
}

type CommandResult struct {
	Case        *domain.RestorationCase       `json:"case"`
	Record      *domain.MonitoringRecord      `json:"record,omitempty"`
	Action      *domain.RemediationAction     `json:"action,omitempty"`
	Certificate *domain.AcceptanceCertificate `json:"certificate,omitempty"`
	Idempotent  bool                          `json:"idempotent"`
}

func (s *Service) CreateCase(command CreateCaseCommand) (CommandResult, error) {
	command.Meta.ExpectedVersion = 0
	if err := command.Meta.validate(domain.RoleMonitor); err != nil {
		return CommandResult{}, err
	}
	if cached, ok := s.cachedCommand(command.Meta.IdempotencyKey); ok {
		return cached, nil
	}
	if existing, ok := s.store.IdempotentCase(command.Meta.IdempotencyKey); ok {
		if !idempotentReceipt(existing, command.Meta.IdempotencyKey, domain.EventCaseCreated) {
			return CommandResult{}, domain.NewError(domain.CodeConflict, "idempotencyKey 已用于其他操作")
		}
		s.rememberCommand(command.Meta.IdempotencyKey, existing.ID)
		return CommandResult{Case: existing, Idempotent: true}, nil
	}
	now := s.now().UTC()
	item, err := domain.NewRestorationCase(s.id("case"), command.Name, strings.ToUpper(command.SiteCode), command.HabitatType, command.Baseline, now)
	if err != nil {
		return CommandResult{}, err
	}
	event, err := domain.NewEvent(s.id("evt"), item.ID, domain.EventCaseCreated, command.Meta.Actor, command.Meta.IdempotencyKey, 1, now, domain.CaseCreatedData{Case: *item})
	if err != nil {
		return CommandResult{}, err
	}
	stored, err := s.store.Append(item.ID, 0, []domain.Event{event})
	if err != nil {
		return CommandResult{}, err
	}
	s.rememberCommand(command.Meta.IdempotencyKey, stored.ID)
	return CommandResult{Case: stored}, nil
}
