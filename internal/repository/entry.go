package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
)

type LedgerEntry struct {
	Sequence     int64        `json:"sequence"`
	PreviousHash string       `json:"previousHash"`
	Hash         string       `json:"hash"`
	Event        domain.Event `json:"event"`
}

type hashMaterial struct {
	Sequence     int64        `json:"sequence"`
	PreviousHash string       `json:"previousHash"`
	Event        domain.Event `json:"event"`
}

func calculateHash(sequence int64, previous string, event domain.Event) (string, error) {
	encoded, err := json.Marshal(hashMaterial{Sequence: sequence, PreviousHash: previous, Event: event})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func newLedgerEntry(sequence int64, previous string, event domain.Event) (LedgerEntry, error) {
	hash, err := calculateHash(sequence, previous, event)
	if err != nil {
		return LedgerEntry{}, err
	}
	return LedgerEntry{Sequence: sequence, PreviousHash: previous, Hash: hash, Event: event}, nil
}
