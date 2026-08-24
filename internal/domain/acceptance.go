package domain

import (
	"strings"
	"time"
)

type AcceptanceDecision string

const (
	DecisionAccepted AcceptanceDecision = "accepted"
	DecisionRejected AcceptanceDecision = "rejected"
)

type AcceptanceCertificate struct {
	ID             string             `json:"id"`
	CaseID         string             `json:"caseId"`
	Reviewer       string             `json:"reviewer"`
	Decision       AcceptanceDecision `json:"decision"`
	ReviewNote     string             `json:"reviewNote"`
	FrozenAt       time.Time          `json:"frozenAt"`
	CredentialCode string             `json:"credentialCode"`
	EvidenceDigest string             `json:"evidenceDigest"`
}

func (certificate AcceptanceCertificate) Validate() error {
	if strings.TrimSpace(certificate.Reviewer) == "" {
		return NewError(CodeInvalid, "独立复核员不能为空")
	}
	if strings.TrimSpace(certificate.ReviewNote) == "" {
		return NewError(CodeInvalid, "复核说明不能为空")
	}
	if certificate.Decision != DecisionAccepted && certificate.Decision != DecisionRejected {
		return NewError(CodeInvalid, "复核决定必须为 accepted 或 rejected")
	}
	return nil
}

type ReviewRecord struct {
	Reviewer   string             `json:"reviewer"`
	Decision   AcceptanceDecision `json:"decision"`
	ReviewNote string             `json:"reviewNote"`
	ReviewedAt time.Time          `json:"reviewedAt"`
}
