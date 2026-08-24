package rules

import (
	"fmt"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
)

func (e *Engine) AssessRetest(habitat string, original domain.MonitoringRecord, observed float64) RetestAssessment {
	current := e.Assess(habitat, original.ExpectedRange, observed)
	oldDistance := distanceFromRange(original.ExpectedRange, original.ObservedValue)
	newDistance := distanceFromRange(original.ExpectedRange, observed)
	improvement := oldDistance - newDistance
	if current.Passed {
		current.RuleHit.Rule = "retest.closed"
		current.RuleHit.Explanation = fmt.Sprintf("复测值 %.2f %s 已回到锁定范围，较原偏差改善 %.2f", observed, original.Unit, improvement)
		current.RuleHit.Suggestion = "整改闭环，可进入独立复核"
		return RetestAssessment{Closed: true, Improvement: improvement, RuleHit: current.RuleHit}
	}
	current.RuleHit.Rule = "retest.still_out_of_range"
	if improvement > 0 {
		current.RuleHit.Explanation = fmt.Sprintf("复测偏差改善 %.2f，但仍未回到锁定范围", improvement)
	} else {
		current.RuleHit.Explanation = fmt.Sprintf("复测未改善，偏差变化 %.2f，整改保持开启", improvement)
	}
	current.RuleHit.Suggestion = "补充整改证据并再次复测"
	return RetestAssessment{Closed: false, Improvement: improvement, RuleHit: current.RuleHit}
}
