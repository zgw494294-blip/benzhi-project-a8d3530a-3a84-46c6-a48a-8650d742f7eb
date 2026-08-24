package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

func DigestText(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func EvidenceDigest(c *RestorationCase) string {
	parts := []string{c.ID, c.BaselineVersion}
	for _, record := range c.Monitoring {
		parts = append(parts, fmt.Sprintf("%s|%s|%.6f|%s|%s", record.ID, record.Indicator, record.ObservedValue, record.EvidenceNote, record.Status))
	}
	for _, action := range c.Remediations {
		parts = append(parts, fmt.Sprintf("%s|%s|%s|%s", action.ID, action.Owner, action.EvidenceNote, action.Status))
	}
	for _, review := range c.Reviews {
		parts = append(parts, fmt.Sprintf("%s|%s|%s|%s", review.Reviewer, review.Decision, review.ReviewNote, review.ReviewedAt.UTC().Format("2006-01-02T15:04:05Z")))
	}
	sort.Strings(parts[2:])
	return DigestText(strings.Join(parts, "\n"))
}

func CredentialCode(caseID, evidenceDigest, reviewer string, frozenAt int64) string {
	raw := fmt.Sprintf("%s|%s|%s|%d", caseID, evidenceDigest, reviewer, frozenAt)
	digest := strings.ToUpper(DigestText(raw))
	return fmt.Sprintf("CCR-%s-%s-%s", digest[:6], digest[6:12], digest[12:18])
}
