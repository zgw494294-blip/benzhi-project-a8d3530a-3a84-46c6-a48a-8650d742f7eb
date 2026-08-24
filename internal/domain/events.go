package domain

import (
	"encoding/json"
	"time"
)

const (
	EventCaseCreated        = "case.created"
	EventMonitoringRecorded = "monitoring.recorded"
	EventRetestSubmitted    = "remediation.retested"
	EventReviewRecorded     = "acceptance.reviewed"
	EventAcceptanceFrozen   = "acceptance.frozen"
)

type Event struct {
	ID             string          `json:"id"`
	CaseID         string          `json:"caseId"`
	Type           string          `json:"type"`
	Version        int64           `json:"version"`
	OccurredAt     time.Time       `json:"occurredAt"`
	Actor          string          `json:"actor"`
	IdempotencyKey string          `json:"idempotencyKey"`
	Data           json.RawMessage `json:"data"`
}

type CaseCreatedData struct {
	Case RestorationCase `json:"case"`
}

type MonitoringRecordedData struct {
	Record      MonitoringRecord   `json:"record"`
	Remediation *RemediationAction `json:"remediation,omitempty"`
}

type RetestSubmittedData struct {
	Record     MonitoringRecord `json:"record"`
	ActionID   string           `json:"actionId"`
	Closed     bool             `json:"closed"`
	Evidence   string           `json:"evidence"`
	ResolvedAt *time.Time       `json:"resolvedAt,omitempty"`
}

type ReviewRecordedData struct {
	Review ReviewRecord `json:"review"`
}

type AcceptanceFrozenData struct {
	Certificate AcceptanceCertificate `json:"certificate"`
}

func NewEvent(id, caseID, eventType, actor, key string, version int64, at time.Time, data any) (Event, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return Event{}, err
	}
	return Event{ID: id, CaseID: caseID, Type: eventType, Actor: actor, IdempotencyKey: key, Version: version, OccurredAt: at.UTC(), Data: payload}, nil
}

func ApplyEvent(c *RestorationCase, event Event) error {
	if c == nil && event.Type != EventCaseCreated {
		return NewError(CodeIntegrity, "事件 %s 缺少案件建档事件", event.ID)
	}
	if c != nil && event.Version != c.Version+1 {
		return NewError(CodeIntegrity, "案件 %s 事件版本不连续", event.CaseID)
	}
	switch event.Type {
	case EventCaseCreated:
		var payload CaseCreatedData
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return err
		}
		*c = payload.Case
	case EventMonitoringRecorded:
		var payload MonitoringRecordedData
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return err
		}
		c.Monitoring = append(c.Monitoring, payload.Record)
		if payload.Remediation != nil {
			c.Remediations = append(c.Remediations, *payload.Remediation)
			c.Status = CaseRemediation
		} else if !c.HasOpenRemediations() {
			c.Status = CaseReadyForReview
		}
	case EventRetestSubmitted:
		var payload RetestSubmittedData
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return err
		}
		c.Monitoring = append(c.Monitoring, payload.Record)
		_, index, ok := c.FindRemediation(payload.ActionID)
		if !ok {
			return NewError(CodeIntegrity, "复测引用的整改 %s 不存在", payload.ActionID)
		}
		c.Remediations[index].Attempts++
		c.Remediations[index].EvidenceNote = payload.Evidence
		if payload.Closed {
			c.Remediations[index].Status = RemediationResolved
			c.Remediations[index].ResolvedAt = payload.ResolvedAt
		}
		if c.HasOpenRemediations() {
			c.Status = CaseRemediation
		} else {
			c.Status = CaseReadyForReview
		}
	case EventReviewRecorded:
		var payload ReviewRecordedData
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return err
		}
		c.Reviews = append(c.Reviews, payload.Review)
		if payload.Review.Decision == DecisionRejected {
			c.Status = CaseReviewRejected
		}
	case EventAcceptanceFrozen:
		var payload AcceptanceFrozenData
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return err
		}
		c.Certificate = &payload.Certificate
		c.Status = CaseAcceptanceFrozen
	default:
		return NewError(CodeIntegrity, "未知领域事件 %s", event.Type)
	}
	if c.ProcessedCommand == nil {
		c.ProcessedCommand = make(map[string]IdempotencyReceipt)
	}
	c.Version = event.Version
	c.UpdatedAt = event.OccurredAt
	if event.IdempotencyKey != "" {
		c.ProcessedCommand[event.IdempotencyKey] = IdempotencyReceipt{Operation: event.Type, Version: event.Version, Resource: event.ID}
	}
	return nil
}
