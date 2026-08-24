package application

import (
	"strings"
	"time"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
)

type SubmitRetestCommand struct {
	Meta          CommandMeta `json:"meta"`
	CaseID        string      `json:"caseId"`
	ActionID      string      `json:"actionId"`
	Owner         string      `json:"owner"`
	ObservedValue float64     `json:"observedValue"`
	EvidenceNote  string      `json:"evidenceNote"`
}

func (s *Service) SubmitRetest(command SubmitRetestCommand) (CommandResult, error) {
	if err := command.Meta.validate(domain.RoleRemediator); err != nil {
		return CommandResult{}, err
	}
	if cached, ok := s.cachedCommand(command.Meta.IdempotencyKey); ok {
		return cached, nil
	}
	item, err := s.store.Get(command.CaseID)
	if err != nil {
		return CommandResult{}, err
	}
	if idempotentReceipt(item, command.Meta.IdempotencyKey, domain.EventRetestSubmitted) {
		return CommandResult{Case: item, Idempotent: true}, nil
	}
	if err := ensureUnusedKey(item, command.Meta.IdempotencyKey, domain.EventRetestSubmitted); err != nil {
		return CommandResult{}, err
	}
	action, _, ok := item.FindRemediation(command.ActionID)
	if !ok {
		return CommandResult{}, domain.NewError(domain.CodeNotFound, "整改任务 %s 不存在", command.ActionID)
	}
	if !action.IsOpen() {
		return CommandResult{}, domain.NewError(domain.CodeInvalidState, "整改任务已关闭")
	}
	owner := strings.TrimSpace(command.Owner)
	if owner == "" {
		owner = command.Meta.Actor
	}
	if !strings.EqualFold(owner, action.Owner) || !strings.EqualFold(command.Meta.Actor, action.Owner) {
		return CommandResult{}, domain.NewError(domain.CodeForbidden, "只有整改责任人 %s 可以提交复测", action.Owner)
	}
	if strings.TrimSpace(command.EvidenceNote) == "" {
		return CommandResult{}, domain.NewError(domain.CodeInvalid, "复测证据说明不能为空")
	}
	original, ok := item.FindMonitoring(action.MonitoringID)
	if !ok {
		return CommandResult{}, domain.NewError(domain.CodeIntegrity, "整改任务缺少原始监测记录")
	}
	assessment := s.rules.AssessRetest(item.HabitatType, original, command.ObservedValue)
	now := s.now().UTC()
	status := domain.MonitoringRetestFailed
	if assessment.Closed {
		status = domain.MonitoringRetestPass
	}
	record := domain.MonitoringRecord{
		ID: s.id("mon"), CaseID: item.ID, Indicator: original.Indicator, ExpectedRange: original.ExpectedRange,
		ObservedValue: command.ObservedValue, Unit: original.Unit, EvidenceNote: strings.TrimSpace(command.EvidenceNote),
		CapturedBy: owner, CapturedAt: now, Status: status, RuleHit: assessment.RuleHit, RetestFor: action.ID,
	}
	var resolvedAt *time.Time
	if assessment.Closed {
		resolvedAt = &now
	}
	payload := domain.RetestSubmittedData{Record: record, ActionID: action.ID, Closed: assessment.Closed, Evidence: record.EvidenceNote, ResolvedAt: resolvedAt}
	event, err := domain.NewEvent(s.id("evt"), item.ID, domain.EventRetestSubmitted, command.Meta.Actor, command.Meta.IdempotencyKey, item.Version+1, now, payload)
	if err != nil {
		return CommandResult{}, err
	}
	stored, err := s.store.Append(item.ID, command.Meta.ExpectedVersion, []domain.Event{event})
	if err != nil {
		return CommandResult{}, err
	}
	s.rememberCommand(command.Meta.IdempotencyKey, stored.ID)
	updated, _, _ := stored.FindRemediation(action.ID)
	return CommandResult{Case: stored, Record: &record, Action: &updated}, nil
}
