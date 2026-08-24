package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
