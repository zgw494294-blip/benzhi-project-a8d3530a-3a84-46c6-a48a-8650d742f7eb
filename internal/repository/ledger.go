package repository

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
)

type Ledger struct {
	path     string
	sequence int64
	lastHash string
}

func openLedger(path string) (*Ledger, []LedgerEntry, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, nil, fmt.Errorf("创建账本目录: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o640)
	if err != nil {
		return nil, nil, fmt.Errorf("打开事件账本: %w", err)
	}
	defer file.Close()
	// 共享锁保证读取期间没有其他实例正在追加，避免读到半写入的行而误判完整性失败。
	if lockErr := syscall.Flock(int(file.Fd()), syscall.LOCK_SH); lockErr != nil {
		return nil, nil, fmt.Errorf("锁定事件账本: %w", lockErr)
	}
	entries, err := readAndValidate(file)
	if unlockErr := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); unlockErr != nil && err == nil {
		err = unlockErr
	}
	if err != nil {
		return nil, nil, err
	}
	ledger := &Ledger{path: path}
	if len(entries) > 0 {
		last := entries[len(entries)-1]
		ledger.sequence = last.Sequence
		ledger.lastHash = last.Hash
	}
	return ledger, entries, nil
}

func readAndValidate(reader io.Reader) ([]LedgerEntry, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	entries := make([]LedgerEntry, 0)
	var previous string
	var sequence int64
	line := 0
	seenEvents := make(map[string]struct{})
	for scanner.Scan() {
		line++
		if len(bytes.TrimSpace(scanner.Bytes())) == 0 {
			continue
		}
		var entry LedgerEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, domain.NewError(domain.CodeIntegrity, "账本第 %d 行不是有效 JSON", line)
		}
		sequence++
		if entry.Sequence != sequence {
			return nil, domain.NewError(domain.CodeIntegrity, "账本序号在第 %d 行不连续", line)
		}
		if entry.PreviousHash != previous {
			return nil, domain.NewError(domain.CodeIntegrity, "账本第 %d 行前序摘要不匹配", line)
		}
		expected, err := calculateHash(entry.Sequence, entry.PreviousHash, entry.Event)
		if err != nil {
			return nil, err
		}
		if entry.Hash != expected {
			return nil, domain.NewError(domain.CodeIntegrity, "账本第 %d 行摘要校验失败", line)
		}
		if _, duplicate := seenEvents[entry.Event.ID]; duplicate {
			return nil, domain.NewError(domain.CodeIntegrity, "账本事件 %s 重复", entry.Event.ID)
		}
		seenEvents[entry.Event.ID] = struct{}{}
		entries = append(entries, entry)
		previous = entry.Hash
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取事件账本: %w", err)
	}
	return entries, nil
}

func (l *Ledger) appendBatch(events []domain.Event) ([]LedgerEntry, error) {
	if len(events) == 0 {
		return nil, domain.NewError(domain.CodeInvalid, "不能追加空事件批次")
	}
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o640)
	if err != nil {
		return nil, fmt.Errorf("打开事件账本进行追加: %w", err)
	}
	// 跨进程互斥：共享同一账本目录的多个实例必须串行追加，否则各自的内存序号会相互覆盖，
	// 导致事件序号/前序摘要链断裂、重启无法通过完整性校验。
	if lockErr := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); lockErr != nil {
		file.Close()
		return nil, fmt.Errorf("锁定事件账本: %w", lockErr)
	}
	defer func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		file.Close()
	}()
	// 在持有跨进程锁的情况下重新读取账本尾部，确认当前磁盘上的真实序号与前序摘要。
	// 其他实例可能在打开本进程账本之后已经追加了事件，必须以磁盘真实尾部为基础续接哈希链。
	sequence, previous, err := readTail(file)
	if err != nil {
		return nil, err
	}
	entries := make([]LedgerEntry, 0, len(events))
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	for _, event := range events {
		sequence++
		entry, err := newLedgerEntry(sequence, previous, event)
		if err != nil {
			return nil, err
		}
		if err := encoder.Encode(entry); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
		previous = entry.Hash
	}
	if _, err = file.Seek(0, io.SeekEnd); err != nil {
		return nil, fmt.Errorf("定位账本末尾: %w", err)
	}
	if _, err = file.Write(buffer.Bytes()); err != nil {
		return nil, fmt.Errorf("追加事件账本: %w", err)
	}
	if err = file.Sync(); err != nil {
		return nil, fmt.Errorf("同步事件账本: %w", err)
	}
	l.sequence = sequence
	l.lastHash = previous
	return entries, nil
}

// readTail 重新读取账本文件以确定当前磁盘上的最后一个序号与前序摘要。
func readTail(file *os.File) (int64, string, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return 0, "", fmt.Errorf("定位账本起始: %w", err)
	}
	entries, err := readAndValidate(file)
	if err != nil {
		return 0, "", err
	}
	if len(entries) == 0 {
		return 0, "", nil
	}
	last := entries[len(entries)-1]
	return last.Sequence, last.Hash, nil
}
