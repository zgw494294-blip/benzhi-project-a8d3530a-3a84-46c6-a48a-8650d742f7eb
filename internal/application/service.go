package application

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/rules"
)

type Service struct {
	store Store
	rules *rules.Engine
	now   func() time.Time
	id    func(string) string
}

func NewService(store Store, engine *rules.Engine) *Service {
	return &Service{
		store: store, rules: engine, now: time.Now, id: randomID,
	}
}

func randomID(prefix string) string {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		return prefix + "-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	return prefix + "-" + hex.EncodeToString(buffer)
}

type CommandMeta struct {
	ExpectedVersion int64       `json:"expectedVersion"`
	IdempotencyKey  string      `json:"idempotencyKey"`
	Actor           string      `json:"actor"`
	Role            domain.Role `json:"role"`
}

func (m CommandMeta) validate(role ...domain.Role) error {
	if strings.TrimSpace(m.IdempotencyKey) == "" {
		return domain.NewError(domain.CodeInvalid, "idempotencyKey 不能为空")
	}
	if len(m.IdempotencyKey) > 128 {
		return domain.NewError(domain.CodeInvalid, "idempotencyKey 不能超过 128 个字符")
	}
	if strings.TrimSpace(m.Actor) == "" {
		return domain.NewError(domain.CodeInvalid, "actor 不能为空")
	}
	if m.ExpectedVersion < 0 {
		return domain.NewError(domain.CodeInvalid, "expectedVersion 不能为负数")
	}
	return domain.RequireRole(m.Role, role...)
}

func idempotentReceipt(item *domain.RestorationCase, key, operation string) bool {
	receipt, ok := item.ProcessedCommand[key]
	if !ok {
		return false
	}
	return receipt.Operation == operation
}

func ensureUnusedKey(item *domain.RestorationCase, key, operation string) error {
	if receipt, ok := item.ProcessedCommand[key]; ok && receipt.Operation != operation {
		return domain.NewError(domain.CodeConflict, "idempotencyKey 已用于其他操作")
	}
	return nil
}
