package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
)

const (
	phaseBefore = uint8(0)
	phaseAfter  = uint8(1)
	recordSize  = 24
)

// JournalRecord 是 journal 文件中的一条固定长度（24 字节）二进制记录。
//
// 字节布局：
//
//	[0:8]   objnum  int64   (little-endian)
//	[8:9]   old_key uint8
//	[9:10]  new_key uint8
//	[10:11] phase   uint8   (0=BEFORE, 1=AFTER)
//	[11:24] padding [13]byte (zeroed)
type JournalRecord struct {
	Objnum  int64
	OldKey  uint8
	NewKey  uint8
	Phase   uint8
	Padding [13]byte
}

// Journal 是追加写入的崩溃恢复日志，保护 KeyMap 写操作的原子性。
type Journal struct {
	mu   sync.Mutex
	file *os.File
	path string
}

// NewJournal 打开或创建 path 处的 journal 文件（追加模式）。
func NewJournal(path string) (*Journal, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open journal %s: %w", path, err)
	}
	return &Journal{file: f, path: path}, nil
}

// WriteBefore 在 PUT 操作开始前写入 BEFORE 记录。
func (j *Journal) WriteBefore(objnum int64, oldKey, newKey uint8) error {
	return j.write(objnum, oldKey, newKey, phaseBefore)
}

// WriteAfter 在 PUT 操作完成后写入 AFTER 记录。
func (j *Journal) WriteAfter(objnum int64, oldKey, newKey uint8) error {
	return j.write(objnum, oldKey, newKey, phaseAfter)
}

// Close 关闭 journal 文件。
func (j *Journal) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.file == nil {
		return nil
	}
	err := j.file.Close()
	j.file = nil
	return err
}

func (j *Journal) write(objnum int64, oldKey, newKey, phase uint8) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.file == nil {
		return fmt.Errorf("journal: write on closed journal")
	}

	var buf [recordSize]byte
	binary.LittleEndian.PutUint64(buf[0:8], uint64(objnum))
	buf[8] = oldKey
	buf[9] = newKey
	buf[10] = phase
	// buf[11:24] stays zero (padding)

	// Each write is followed by fsync for crash durability.
	// This adds per-write latency; if throughput matters more than safety,
	// batching writes before syncing can be considered.
	if _, err := j.file.Write(buf[:]); err != nil {
		return fmt.Errorf("journal: write record: %w", err)
	}
	if err := j.file.Sync(); err != nil {
		return fmt.Errorf("journal: sync: %w", err)
	}
	return nil
}

// Recover 扫描 jnlPath 处的 journal 文件，按 objnum 分组取最后一条记录，
// 根据 phase 将 KeyMap 中的 busy 对象恢复到正确状态，
// 然后截断 journal 文件（恢复完成，重新开始）。
func Recover(km *KeyMap, jnlPath string) error {
	if jnlPath == "" {
		return nil
	}
	data, err := os.ReadFile(jnlPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // normal: first startup, no journal yet
		}
		return fmt.Errorf("journal: read %s: %w", jnlPath, err)
	}

	// 解析所有记录，按 objnum 保留最后一条
	last := make(map[int64]JournalRecord)
	total := len(data) / recordSize
	for i := 0; i < total; i++ {
		rec := parseRecord(data[i*recordSize : (i+1)*recordSize])
		last[rec.Objnum] = rec
	}

	km.mu.Lock()
	defer km.mu.Unlock()

	// 根据最后一条记录恢复每个 busy 对象
	for objnum, rec := range last {
		if objnum < 0 || objnum >= int64(len(km.data)) {
			continue
		}
		if km.data[objnum]&0x80 == 0 {
			// 不处于 busy 状态，跳过
			continue
		}
		if rec.Phase == phaseBefore {
			// PUT 被中断：恢复为 old_key
			km.data[objnum] = rec.OldKey & 0x7F
		} else {
			// PUT 已完成：更新为 new_key
			km.data[objnum] = rec.NewKey & 0x7F
		}
	}

	// 截断 journal 文件（恢复完成）
	if err := os.Truncate(jnlPath, 0); err != nil {
		return fmt.Errorf("failed to truncate journal %s: %w", jnlPath, err)
	}
	return nil
}

// RecoverBusy 在启动时处理 KeyMap 中残留的 busy 条目。
// 如果提供了 jnlPath，则调用 Recover 进行日志恢复；
// 否则仅对每个 busy 条目打印警告，并将其重置为 UNKNOWN (0)。
func RecoverBusy(km *KeyMap, jnlPath string) {
	if jnlPath != "" {
		if err := Recover(km, jnlPath); err != nil {
			log.Printf("WARNING: journal recovery failed: %v", err)
		}
		return
	}

	// 无 journal：清除 busy 标志并设置为 UNKNOWN(0)，仅记录告警
	km.mu.Lock()
	defer km.mu.Unlock()
	for i, b := range km.data {
		if b&0x80 != 0 {
			log.Printf("WARNING: object %d has busy flag set with no journal — marking as UNKNOWN", i)
			km.data[i] = 0
		}
	}
}

func parseRecord(buf []byte) JournalRecord {
	return JournalRecord{
		Objnum: int64(binary.LittleEndian.Uint64(buf[0:8])),
		OldKey: buf[8],
		NewKey: buf[9],
		Phase:  buf[10],
	}
}
