package domain

import (
	"strings"
	"time"
)

type MonitoringStatus string

const (
	MonitoringPass                MonitoringStatus = "pass"
	MonitoringRemediationRequired MonitoringStatus = "remediation_required"
	MonitoringRetestPass          MonitoringStatus = "retest_pass"
	MonitoringRetestFailed        MonitoringStatus = "retest_failed"
)

type RuleHit struct {
	Rule        string `json:"rule"`
	RiskLevel   string `json:"riskLevel"`
	Explanation string `json:"explanation"`
	Suggestion  string `json:"suggestion"`
}

type MonitoringRecord struct {
	ID            string           `json:"id"`
	CaseID        string           `json:"caseId"`
	Indicator     string           `json:"indicator"`
	ExpectedRange IndicatorRange   `json:"expectedRange"`
	ObservedValue float64          `json:"observedValue"`
	Unit          string           `json:"unit"`
	EvidenceNote  string           `json:"evidenceNote"`
	CapturedBy    string           `json:"capturedBy"`
	CapturedAt    time.Time        `json:"capturedAt"`
	Status        MonitoringStatus `json:"status"`
	RuleHit       RuleHit          `json:"ruleHit"`
	RetestFor     string           `json:"retestFor,omitempty"`
}

func (record MonitoringRecord) Validate() error {
	if strings.TrimSpace(record.Indicator) == "" {
		return NewError(CodeInvalid, "监测指标不能为空")
	}
	if strings.TrimSpace(record.EvidenceNote) == "" {
		return NewError(CodeInvalid, "证据说明不能为空")
	}
	if strings.TrimSpace(record.CapturedBy) == "" {
		return NewError(CodeInvalid, "采集人不能为空")
	}
	if record.CapturedAt.IsZero() {
		return NewError(CodeInvalid, "采集时间不能为空")
	}
	if record.Unit != record.ExpectedRange.Unit {
		return NewError(CodeInvalid, "指标 %s 的单位必须为 %s", record.Indicator, record.ExpectedRange.Unit)
	}
	return nil
}
