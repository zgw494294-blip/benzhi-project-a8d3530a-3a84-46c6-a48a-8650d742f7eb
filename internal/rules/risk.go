package rules

import (
	"fmt"
	"math"
	"strings"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
)

func (e *Engine) Assess(habitat string, expected domain.IndicatorRange, observed float64) Assessment {
	key := assessmentCacheKey{
		habitat: habitat, indicator: expected.Indicator, minimum: expected.Minimum,
		maximum: expected.Maximum, unit: expected.Unit, observed: observed,
	}
	if cached, ok := e.lookupAssessment(key); ok {
		return cached
	}
	result := e.assess(habitat, expected, observed)
	e.rememberAssessment(key, result)
	return result
}

func (e *Engine) assess(habitat string, expected domain.IndicatorRange, observed float64) Assessment {
	distance := distanceFromRange(expected, observed)
	if distance == 0 {
		return Assessment{Passed: true, RuleHit: domain.RuleHit{
			Rule: "baseline.in_range", RiskLevel: "none",
			Explanation: fmt.Sprintf("观测值 %.2f %s 位于锁定范围 %.2f–%.2f 内", observed, expected.Unit, expected.Minimum, expected.Maximum),
			Suggestion:  "保持当前措施并按计划持续监测",
		}}
	}
	span := math.Abs(expected.Maximum - expected.Minimum)
	if span < 0.000001 {
		span = math.Max(math.Abs(expected.Maximum), 1)
	}
	relative := distance / span
	factor := habitatSensitivity(habitat)
	adjusted := relative * factor
	risk := "low"
	if adjusted > 0.25 {
		risk = "high"
	} else if adjusted > 0.10 {
		risk = "medium"
	}
	direction := "低于"
	if observed > expected.Maximum {
		direction = "高于"
	}
	return Assessment{
		Passed: false, Distance: distance, RelativeGap: relative,
		RuleHit: domain.RuleHit{
			Rule: "baseline.out_of_range", RiskLevel: risk,
			Explanation: fmt.Sprintf("观测值 %.2f %s %s锁定范围，偏差 %.2f；%s敏感系数 %.2f", observed, expected.Unit, direction, distance, habitatLabel(habitat), factor),
			Suggestion:  remediationSuggestion(expected.Indicator, direction, risk),
		},
	}
}

func distanceFromRange(expected domain.IndicatorRange, value float64) float64 {
	if value < expected.Minimum {
		return expected.Minimum - value
	}
	if value > expected.Maximum {
		return value - expected.Maximum
	}
	return 0
}

func habitatSensitivity(habitat string) float64 {
	switch strings.ToLower(strings.TrimSpace(habitat)) {
	case "coral", "珊瑚礁":
		return 1.35
	case "mangrove", "红树林":
		return 1.15
	case "saltmarsh", "盐沼":
		return 1.05
	case "seagrass", "海草床":
		return 1.25
	default:
		return 1
	}
}

func habitatLabel(habitat string) string {
	if strings.TrimSpace(habitat) == "" {
		return "该栖息地"
	}
	return strings.TrimSpace(habitat)
}

func remediationSuggestion(indicator, direction, risk string) string {
	urgency := "纳入常规整改"
	if risk == "medium" {
		urgency = "优先排查来源并在期限内复测"
	}
	if risk == "high" {
		urgency = "立即控制风险源并加密采样"
	}
	return fmt.Sprintf("%s：%s指标%s基线", urgency, indicator, direction)
}
