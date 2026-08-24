package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
)

type snapshotFile struct {
	FormatVersion int         `json:"formatVersion"`
	Projection    *projection `json:"projection"`
}

func saveSnapshot(path string, p *projection) error {
	encoded, err := json.MarshalIndent(snapshotFile{FormatVersion: 1, Projection: p}, "", "  ")
	if err != nil {
		return fmt.Errorf("编码投影快照: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".snapshot-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时快照: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err = temporary.Chmod(0o640); err == nil {
		_, err = temporary.Write(encoded)
	}
	if err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("写入投影快照: %w", err)
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return fmt.Errorf("替换投影快照: %w", err)
	}
	return nil
}

func loadSnapshot(path string) (*snapshotFile, error) {
	encoded, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var snapshot snapshotFile
	if err := json.Unmarshal(encoded, &snapshot); err != nil {
		return nil, fmt.Errorf("解析投影快照: %w", err)
	}
	if snapshot.FormatVersion != 1 || snapshot.Projection == nil {
		return nil, fmt.Errorf("不支持的投影快照格式")
	}
	return &snapshot, nil
}

func recoverProjection(path string, entries []LedgerEntry) (*projection, error) {
	snapshot, err := loadSnapshot(path)
	if err != nil {
		return nil, fmt.Errorf("加载投影快照: %w", err)
	}
	if snapshot == nil {
		p, replayErr := replay(entries)
		if replayErr != nil {
			return nil, fmt.Errorf("重放事件账本: %w", replayErr)
		}
		return p, nil
	}
	sequence := snapshot.Projection.LastSequence
	if sequence > int64(len(entries)) {
		return nil, domain.NewError(domain.CodeIntegrity, "快照序号超前于事件账本")
	}
	if sequence > 0 && entries[sequence-1].Hash != snapshot.Projection.LastHash {
		return nil, domain.NewError(domain.CodeIntegrity, "快照摘要与事件账本不匹配")
	}
	return snapshot.Projection, nil
}
