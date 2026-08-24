package domain

import (
	"strings"
	"time"
)

type CaseStatus string

const (
	CaseMonitoring       CaseStatus = "monitoring"
	CaseRemediation      CaseStatus = "remediation"
	CaseReadyForReview   CaseStatus = "ready_for_review"
	CaseReviewRejected   CaseStatus = "review_rejected"
	CaseAcceptanceFrozen CaseStatus = "acceptance_frozen"
)

type RestorationCase struct {
	ID               string                        `json:"id"`
	Name             string                        `json:"name"`
	SiteCode         string                        `json:"siteCode"`
	HabitatType      string                        `json:"habitatType"`
	BaselineVersion  string                        `json:"baselineVersion"`
	Baseline         map[string]IndicatorRange     `json:"baseline"`
	Status           CaseStatus                    `json:"status"`
	Version          int64                         `json:"version"`
	CreatedAt        time.Time                     `json:"createdAt"`
	UpdatedAt        time.Time                     `json:"updatedAt"`
	Monitoring       []MonitoringRecord            `json:"monitoring"`
	Remediations     []RemediationAction           `json:"remediations"`
	Reviews          []ReviewRecord                `json:"reviews"`
	Certificate      *AcceptanceCertificate        `json:"certificate,omitempty"`
	ProcessedCommand map[string]IdempotencyReceipt `json:"processedCommand,omitempty"`
}

type IdempotencyReceipt struct {
	Operation string `json:"operation"`
	Version   int64  `json:"version"`
	Resource  string `json:"resource"`
}

func NewRestorationCase(id, name, siteCode, habitat string, ranges []IndicatorRange, now time.Time) (*RestorationCase, error) {
	name = strings.TrimSpace(name)
	siteCode = strings.TrimSpace(siteCode)
	habitat = strings.TrimSpace(habitat)
	if id == "" || name == "" || siteCode == "" || habitat == "" {
		return nil, NewError(CodeInvalid, "案件名称、站点编码和栖息地类型不能为空")
	}
	baseline, err := NormalizeBaseline(ranges)
	if err != nil {
		return nil, err
	}
	return &RestorationCase{
		ID:               id,
		Name:             name,
		SiteCode:         siteCode,
		HabitatType:      habitat,
		BaselineVersion:  BaselineVersion(baseline),
		Baseline:         baseline,
		Status:           CaseMonitoring,
		CreatedAt:        now.UTC(),
		UpdatedAt:        now.UTC(),
		ProcessedCommand: make(map[string]IdempotencyReceipt),
	}, nil
}

func (c *RestorationCase) BaselineFor(indicator string) (IndicatorRange, error) {
	rangeValue, ok := c.Baseline[strings.ToLower(strings.TrimSpace(indicator))]
	if !ok {
		return IndicatorRange{}, NewError(CodeInvalid, "指标 %s 不在锁定基线中", indicator)
	}
	return rangeValue, nil
}

func (c *RestorationCase) FindMonitoring(id string) (MonitoringRecord, bool) {
	for _, record := range c.Monitoring {
		if record.ID == id {
			return record, true
		}
	}
	return MonitoringRecord{}, false
}

func (c *RestorationCase) FindRemediation(id string) (RemediationAction, int, bool) {
	for index, action := range c.Remediations {
		if action.ID == id {
			return action, index, true
		}
	}
	return RemediationAction{}, -1, false
}

func (c *RestorationCase) HasOpenRemediations() bool {
	for _, action := range c.Remediations {
		if action.IsOpen() {
			return true
		}
	}
	return false
}

func (c *RestorationCase) CapturedBy(person string) bool {
	for _, record := range c.Monitoring {
		if strings.EqualFold(strings.TrimSpace(record.CapturedBy), strings.TrimSpace(person)) {
			return true
		}
	}
	return false
}

func (c *RestorationCase) CanReview(reviewer string) error {
	if c.Status == CaseAcceptanceFrozen {
		return NewError(CodeInvalidState, "验收已冻结，不能再次复核")
	}
	if len(c.Monitoring) == 0 {
		return NewError(CodeInvalidState, "至少需要一条监测证据才能复核")
	}
	if c.HasOpenRemediations() {
		return NewError(CodeInvalidState, "仍有未关闭整改，不能冻结验收")
	}
	if c.CapturedBy(reviewer) {
		return NewError(CodeForbidden, "独立复核员不能是证据采集人")
	}
	return nil
}

func (c *RestorationCase) Clone() *RestorationCase {
	clone := *c
	clone.Baseline = CloneBaseline(c.Baseline)
	clone.Monitoring = append([]MonitoringRecord(nil), c.Monitoring...)
	clone.Remediations = append([]RemediationAction(nil), c.Remediations...)
	clone.Reviews = append([]ReviewRecord(nil), c.Reviews...)
	clone.ProcessedCommand = make(map[string]IdempotencyReceipt, len(c.ProcessedCommand))
	for key, value := range c.ProcessedCommand {
		clone.ProcessedCommand[key] = value
	}
	if c.Certificate != nil {
		certificate := *c.Certificate
		clone.Certificate = &certificate
	}
	return &clone
}
