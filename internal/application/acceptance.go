package application

import (
	"strings"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
)

type SubmitAcceptanceCommand struct {
	Meta       CommandMeta               `json:"meta"`
	CaseID     string                    `json:"caseId"`
	Reviewer   string                    `json:"reviewer"`
	Decision   domain.AcceptanceDecision `json:"decision"`
	ReviewNote string                    `json:"reviewNote"`
}

func (s *Service) SubmitAcceptance(command SubmitAcceptanceCommand) (CommandResult, error) {
	if err := command.Meta.validate(domain.RoleReviewer); err != nil {
		return CommandResult{}, err
	}
	item, err := s.store.Get(command.CaseID)
	if err != nil {
		return CommandResult{}, err
	}
	terminalOperation := domain.EventReviewRecorded
	if command.Decision == domain.DecisionAccepted {
		terminalOperation = domain.EventAcceptanceFrozen
	}
	if idempotentReceipt(item, command.Meta.IdempotencyKey, terminalOperation) {
		return CommandResult{Case: item, Certificate: item.Certificate, Idempotent: true}, nil
	}
	if err := ensureUnusedKey(item, command.Meta.IdempotencyKey, terminalOperation); err != nil {
		return CommandResult{}, err
	}
	reviewer := strings.TrimSpace(command.Reviewer)
	if reviewer == "" {
		reviewer = command.Meta.Actor
	}
	if !strings.EqualFold(reviewer, command.Meta.Actor) {
		return CommandResult{}, domain.NewError(domain.CodeForbidden, "复核员身份必须与 actor 一致")
	}
	if err := item.CanReview(reviewer); err != nil {
		return CommandResult{}, err
	}
	now := s.now().UTC()
	review := domain.ReviewRecord{Reviewer: reviewer, Decision: command.Decision, ReviewNote: strings.TrimSpace(command.ReviewNote), ReviewedAt: now}
	check := domain.AcceptanceCertificate{Reviewer: reviewer, Decision: command.Decision, ReviewNote: review.ReviewNote}
	if err := check.Validate(); err != nil {
		return CommandResult{}, err
	}
	reviewEvent, err := domain.NewEvent(s.id("evt"), item.ID, domain.EventReviewRecorded, command.Meta.Actor, command.Meta.IdempotencyKey, item.Version+1, now, domain.ReviewRecordedData{Review: review})
	if err != nil {
		return CommandResult{}, err
	}
	events := []domain.Event{reviewEvent}
	var certificate *domain.AcceptanceCertificate
	if command.Decision == domain.DecisionAccepted {
		withReview := item.Clone()
		withReview.Reviews = append(withReview.Reviews, review)
		digest := domain.EvidenceDigest(withReview)
		certificate = &domain.AcceptanceCertificate{
			ID: s.id("cert"), CaseID: item.ID, Reviewer: reviewer, Decision: command.Decision,
			ReviewNote: review.ReviewNote, FrozenAt: now, EvidenceDigest: digest,
		}
		certificate.CredentialCode = domain.CredentialCode(item.ID, digest, reviewer, now.UnixNano())
		frozenEvent, eventErr := domain.NewEvent(s.id("evt"), item.ID, domain.EventAcceptanceFrozen, command.Meta.Actor, command.Meta.IdempotencyKey, item.Version+2, now, domain.AcceptanceFrozenData{Certificate: *certificate})
		if eventErr != nil {
			return CommandResult{}, eventErr
		}
		events = append(events, frozenEvent)
	}
	expectedVersion := command.Meta.ExpectedVersion
	if certificate != nil {
		reviewed, appendErr := s.store.Append(item.ID, expectedVersion, events[:1])
		if appendErr != nil {
			return CommandResult{}, appendErr
		}
		expectedVersion = reviewed.Version
		events = events[1:]
	}
	stored, err := s.store.Append(item.ID, expectedVersion, events)
	if err != nil {
		return CommandResult{}, err
	}
	return CommandResult{Case: stored, Certificate: certificate}, nil
}
