package application

import (
	"strings"
	"time"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
)

type SubmitMonitoringCommand struct {
	Meta             CommandMeta `json:"meta"`
	CaseID           string      `json:"caseId"`
	Indicator        string      `json:"indicator"`
	ObservedValue    float64     `json:"observedValue"`
	Unit             string      `json:"unit"`
	EvidenceNote     string      `json:"evidenceNote"`
	CapturedBy       string      `json:"capturedBy"`
	CapturedAt       time.Time   `json:"capturedAt"`
	RemediationOwner string      `json:"remediationOwner"`
	RemediationDueAt time.Time   `json:"remediationDueAt"`
}

func (s *Service) SubmitMonitoring(command SubmitMonitoringCommand) (CommandResult, error) {
	if err := command.Meta.validate(domain.RoleMonitor); err != nil {
		return CommandResult{}, err
	}
	if cached, ok := s.cachedCommand(command.Meta.IdempotencyKey); ok {
		return cached, nil
	}
	item, err := s.store.Get(command.CaseID)
	if err != nil {
		return CommandResult{}, err
	}
	if idempotentReceipt(item, command.Meta.IdempotencyKey, domain.EventMonitoringRecorded) {
		return CommandResult{Case: item, Idempotent: true}, nil
	}
	if err := ensureUnusedKey(item, command.Meta.IdempotencyKey, domain.EventMonitoringRecorded); err != nil {
		return CommandResult{}, err
	}
	if item.Status == domain.CaseAcceptanceFrozen {
		return CommandResult{}, domain.NewError(domain.CodeInvalidState, "验收冻结后不能新增监测证据")
	}
	expected, err := item.BaselineFor(command.Indicator)
	if err != nil {
		return CommandResult{}, err
	}
	now := s.now().UTC()
	capturedAt := command.CapturedAt.UTC()
	if capturedAt.IsZero() {
		capturedAt = now
	}
	if capturedAt.After(now.Add(5 * time.Minute)) {
		return CommandResult{}, domain.NewError(domain.CodeInvalid, "采集时间不能晚于当前时间五分钟以上")
	}
	assessment := s.rules.Assess(item.HabitatType, expected, command.ObservedValue)
	record := domain.MonitoringRecord{
		ID: s.id("mon"), CaseID: item.ID, Indicator: expected.Indicator, ExpectedRange: expected,
		ObservedValue: command.ObservedValue, Unit: strings.TrimSpace(command.Unit), EvidenceNote: strings.TrimSpace(command.EvidenceNote),
		CapturedBy: strings.TrimSpace(command.CapturedBy), CapturedAt: capturedAt, RuleHit: assessment.RuleHit,
		Status: domain.MonitoringPass,
	}
	if record.CapturedBy == "" {
		record.CapturedBy = command.Meta.Actor
	}
	if err := record.Validate(); err != nil {
		return CommandResult{}, err
	}
	var action *domain.RemediationAction
	if !assessment.Passed {
		record.Status = domain.MonitoringRemediationRequired
		action = &domain.RemediationAction{
			ID: s.id("rem"), CaseID: item.ID, MonitoringID: record.ID, IssueType: assessment.RuleHit.RiskLevel,
			Action: assessment.RuleHit.Suggestion, Owner: strings.TrimSpace(command.RemediationOwner), DueAt: command.RemediationDueAt.UTC(), Status: domain.RemediationOpen,
		}
		if err := action.ValidateNew(now); err != nil {
			return CommandResult{}, err
		}
	}
	event, err := domain.NewEvent(s.id("evt"), item.ID, domain.EventMonitoringRecorded, command.Meta.Actor, command.Meta.IdempotencyKey, item.Version+1, now, domain.MonitoringRecordedData{Record: record, Remediation: action})
	if err != nil {
		return CommandResult{}, err
	}
	stored, err := s.store.Append(item.ID, command.Meta.ExpectedVersion, []domain.Event{event})
	if err != nil {
		return CommandResult{}, err
	}
	s.rememberCommand(command.Meta.IdempotencyKey, stored.ID)
	return CommandResult{Case: stored, Record: &record, Action: action}, nil
}
