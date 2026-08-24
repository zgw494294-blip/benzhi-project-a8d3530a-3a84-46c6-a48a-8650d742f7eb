package domain

import (
	"strings"
	"time"
)

type RemediationStatus string

const (
	RemediationOpen     RemediationStatus = "open"
	RemediationResolved RemediationStatus = "resolved"
)

type RemediationAction struct {
	ID           string            `json:"id"`
	CaseID       string            `json:"caseId"`
	MonitoringID string            `json:"monitoringID"`
	IssueType    string            `json:"issueType"`
	Action       string            `json:"action"`
	Owner        string            `json:"owner"`
	DueAt        time.Time         `json:"dueAt"`
	EvidenceNote string            `json:"evidenceNote"`
	ResolvedAt   *time.Time        `json:"resolvedAt,omitempty"`
	Status       RemediationStatus `json:"status"`
	Attempts     int               `json:"attempts"`
}

func (action RemediationAction) ValidateNew(now time.Time) error {
	if strings.TrimSpace(action.Owner) == "" {
		return NewError(CodeInvalid, "整改责任人不能为空")
	}
	if strings.TrimSpace(action.Action) == "" {
		return NewError(CodeInvalid, "整改措施不能为空")
	}
	if !action.DueAt.After(now) {
		return NewError(CodeInvalid, "整改期限必须晚于当前时间")
	}
	return nil
}

func (action RemediationAction) IsOpen() bool {
	return action.Status == RemediationOpen
}
