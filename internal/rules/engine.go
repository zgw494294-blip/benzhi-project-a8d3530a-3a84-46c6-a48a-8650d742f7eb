package rules

import (
	"sync"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
)

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

type Engine struct {
	mu          sync.RWMutex
	assessments map[assessmentCacheKey]Assessment
}

func NewEngine() *Engine { return &Engine{assessments: make(map[assessmentCacheKey]Assessment)} }

// cachedAssessment returns the cached assessment for the given key, if any.
// Callers must hold either the read or write lock via lookupAssessment.
func (e *Engine) cachedAssessment(key assessmentCacheKey) (Assessment, bool) {
	result, ok := e.assessments[key]
	return result, ok
}

func (e *Engine) lookupAssessment(key assessmentCacheKey) (Assessment, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cachedAssessment(key)
}

// rememberAssessment stores a freshly computed assessment while avoiding
// duplicate entries under concurrent callers. It performs a double-check:
// after acquiring the write lock it re-examines the cache so that a concurrent
// caller that already stored the same key wins and the cache stays consistent.
func (e *Engine) rememberAssessment(key assessmentCacheKey, result Assessment) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.assessments[key]; ok {
		return
	}
	e.assessments[key] = result
}
