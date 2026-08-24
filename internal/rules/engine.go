package rules

import "benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"

type Assessment struct {
	Passed      bool           `json:"passed"`
	Distance    float64        `json:"distance"`
	RelativeGap float64        `json:"relativeGap"`
	RuleHit     domain.RuleHit `json:"ruleHit"`
}

type RetestAssessment struct {
	Closed      bool           `json:"closed"`
	Improvement float64        `json:"improvement"`
	RuleHit     domain.RuleHit `json:"ruleHit"`
}

type Engine struct{}

func NewEngine() *Engine { return &Engine{} }
