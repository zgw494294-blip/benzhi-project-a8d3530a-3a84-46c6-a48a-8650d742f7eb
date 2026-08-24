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

type assessmentCacheKey struct {
	habitat   string
	indicator string
	minimum   float64
	maximum   float64
	unit      string
	observed  float64
}

type assessmentCacheEntry struct {
	key    assessmentCacheKey
	result Assessment
}

type Engine struct {
	assessmentCache []assessmentCacheEntry
}

func NewEngine() *Engine { return &Engine{} }

func (e *Engine) cachedAssessment(key assessmentCacheKey) (Assessment, bool) {
	for _, entry := range e.assessmentCache {
		if entry.key == key {
			return entry.result, true
		}
	}
	return Assessment{}, false
}

func (e *Engine) rememberAssessment(key assessmentCacheKey, result Assessment) {
	e.assessmentCache = append(e.assessmentCache, assessmentCacheEntry{key: key, result: result})
}
