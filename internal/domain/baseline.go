package domain

import (
	"fmt"
	"sort"
	"strings"
)

type IndicatorRange struct {
	Indicator string  `json:"indicator"`
	Minimum   float64 `json:"minimum"`
	Maximum   float64 `json:"maximum"`
	Unit      string  `json:"unit"`
}

func (r IndicatorRange) Validate() error {
	if strings.TrimSpace(r.Indicator) == "" {
		return NewError(CodeInvalid, "指标名称不能为空")
	}
	if strings.TrimSpace(r.Unit) == "" {
		return NewError(CodeInvalid, "指标 %s 的单位不能为空", r.Indicator)
	}
	if r.Minimum > r.Maximum {
		return NewError(CodeInvalid, "指标 %s 的最小值不能大于最大值", r.Indicator)
	}
	return nil
}

func NormalizeBaseline(ranges []IndicatorRange) (map[string]IndicatorRange, error) {
	if len(ranges) == 0 {
		return nil, NewError(CodeInvalid, "至少需要一个基线指标")
	}
	result := make(map[string]IndicatorRange, len(ranges))
	for _, item := range ranges {
		item.Indicator = strings.TrimSpace(item.Indicator)
		item.Unit = strings.TrimSpace(item.Unit)
		if err := item.Validate(); err != nil {
			return nil, err
		}
		key := strings.ToLower(item.Indicator)
		if _, exists := result[key]; exists {
			return nil, NewError(CodeInvalid, "基线指标 %s 重复", item.Indicator)
		}
		result[key] = item
	}
	return result, nil
}

func BaselineVersion(ranges map[string]IndicatorRange) string {
	keys := make([]string, 0, len(ranges))
	for key := range ranges {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, key := range keys {
		r := ranges[key]
		fmt.Fprintf(&b, "%s:%.6f:%.6f:%s;", key, r.Minimum, r.Maximum, r.Unit)
	}
	return DigestText(b.String())[:16]
}

func CloneBaseline(source map[string]IndicatorRange) map[string]IndicatorRange {
	copyMap := make(map[string]IndicatorRange, len(source))
	for key, value := range source {
		copyMap[key] = value
	}
	return copyMap
}
