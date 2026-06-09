package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
)

// Task 3.1: WriteJournal BEFORE 记录正确追加到文件
func TestWriteJournal_Before_AppendsRecord(t *testing.T) {
	path := t.TempDir() + "/journal.bin"
	j, err := NewJournal(path)
	if err != nil {
		t.Fatalf("NewJournal failed: %v", err)
	}

	if err := j.WriteBefore(42, 3, 4); err != nil {
		t.Fatalf("WriteBefore failed: %v", err)
	}
	if err := j.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if len(data) != recordSize {
		t.Fatalf("expected file size %d, got %d", recordSize, len(data))
	}

	objnum := int64(binary.LittleEndian.Uint64(data[0:8]))
	if objnum != 42 {
		t.Errorf("objnum: expected 42, got %d", objnum)
	}
	if data[8] != 3 {
		t.Errorf("old_key: expected 3, got %d", data[8])
	}
	if data[9] != 4 {
		t.Errorf("new_key: expected 4, got %d", data[9])
	}
	if data[10] != phaseBefore {
		t.Errorf("phase: expected phaseBefore(%d), got %d", phaseBefore, data[10])
	}
	// padding must be zero
	if !bytes.Equal(data[11:24], make([]byte, 13)) {
		t.Errorf("padding bytes are not zero: %v", data[11:24])
	}
}

// Task 3.3: WriteJournal AFTER 记录正确追加到文件
func TestWriteJournal_After_AppendsRecord(t *testing.T) {
	path := t.TempDir() + "/journal.bin"
	j, err := NewJournal(path)
	if err != nil {
		t.Fatalf("NewJournal failed: %v", err)
	}

	if err := j.WriteAfter(7, 1, 2); err != nil {
		t.Fatalf("WriteAfter failed: %v", err)
	}
	if err := j.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if len(data) != recordSize {
		t.Fatalf("expected file size %d, got %d", recordSize, len(data))
	}

	objnum := int64(binary.LittleEndian.Uint64(data[0:8]))
	if objnum != 7 {
		t.Errorf("objnum: expected 7, got %d", objnum)
	}
	if data[8] != 1 {
		t.Errorf("old_key: expected 1, got %d", data[8])
	}
	if data[9] != 2 {
		t.Errorf("new_key: expected 2, got %d", data[9])
	}
	if data[10] != phaseAfter {
		t.Errorf("phase: expected phaseAfter(%d), got %d", phaseAfter, data[10])
	}
	if !bytes.Equal(data[11:24], make([]byte, 13)) {
		t.Errorf("padding bytes are not zero: %v", data[11:24])
	}
}

// Task 3.5: Recover 仅有 BEFORE 记录时恢复为 old_key
func TestRecover_OnlyBefore_RestoresOldKey(t *testing.T) {
	tmpDir := t.TempDir()
	jPath := tmpDir + "/journal.bin"
	kmPath := tmpDir + "/keymap.dat"

	// 构建 keymap：objnum=5 处于 busy 状态（old_key=3，写过程中断）
	km, err := NewKeyMap(kmPath, 10)
	if err != nil {
		t.Fatalf("NewKeyMap failed: %v", err)
	}
	km.data[5] = 0x83 // busy + old_key=3

	// 写入一条 BEFORE 记录：objnum=5, old_key=3, new_key=4
	j, err := NewJournal(jPath)
	if err != nil {
		t.Fatalf("NewJournal failed: %v", err)
	}
	if err := j.WriteBefore(5, 3, 4); err != nil {
		t.Fatalf("WriteBefore failed: %v", err)
	}
	j.Close()

	// 执行恢复
	if err := Recover(km, jPath); err != nil {
		t.Fatalf("Recover failed: %v", err)
	}

	// 期望：busy 被清除，key 恢复为 old_key=3
	if km.data[5] != 3 {
		t.Errorf("expected data[5]=3 (old_key restored), got %d", km.data[5])
	}

	// journal 文件应被截断为 0
	fi, err := os.Stat(jPath)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if fi.Size() != 0 {
		t.Errorf("expected journal truncated to 0, got %d", fi.Size())
	}
}

// Task 3.7: Recover 有 AFTER 记录时恢复为 new_key
func TestRecover_WithAfter_UpdatesToNewKey(t *testing.T) {
	tmpDir := t.TempDir()
	jPath := tmpDir + "/journal.bin"
	kmPath := tmpDir + "/keymap.dat"

	// 构建 keymap：objnum=2 处于 busy 状态（old_key=1）
	km, err := NewKeyMap(kmPath, 5)
	if err != nil {
		t.Fatalf("NewKeyMap failed: %v", err)
	}
	km.data[2] = 0x81 // busy + old_key=1

	// 写入 BEFORE 和 AFTER 记录：objnum=2, old_key=1, new_key=2
	j, err := NewJournal(jPath)
	if err != nil {
		t.Fatalf("NewJournal failed: %v", err)
	}
	if err := j.WriteBefore(2, 1, 2); err != nil {
		t.Fatalf("WriteBefore failed: %v", err)
	}
	if err := j.WriteAfter(2, 1, 2); err != nil {
		t.Fatalf("WriteAfter failed: %v", err)
	}
	j.Close()

	// 执行恢复
	if err := Recover(km, jPath); err != nil {
		t.Fatalf("Recover failed: %v", err)
	}

	// 期望：busy 被清除，key 更新为 new_key=2
	if km.data[2] != 2 {
		t.Errorf("expected data[2]=2 (new_key), got %d", km.data[2])
	}

	// journal 文件应被截断为 0
	fi, err := os.Stat(jPath)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if fi.Size() != 0 {
		t.Errorf("expected journal truncated to 0, got %d", fi.Size())
	}
}

// Task 3.9: 无 journal 时 busy 对象保持 UNKNOWN（仅告警）
func TestRecoverBusy_NoJournal_BusyRemainsUnknown(t *testing.T) {
	kmPath := t.TempDir() + "/keymap.dat"

	km, err := NewKeyMap(kmPath, 4)
	if err != nil {
		t.Fatalf("NewKeyMap failed: %v", err)
	}
	// 设置两个 busy 对象
	km.data[1] = 0x82 // busy + old_key=2
	km.data[3] = 0x85 // busy + old_key=5

	// 不提供 journal 路径（空字符串）
	RecoverBusy(km, "")

	// busy 标志应被清除，值变为 0（UNKNOWN）
	if km.data[1]&0x80 != 0 {
		t.Errorf("expected busy bit cleared for data[1], got 0x%02x", km.data[1])
	}
	if km.data[3]&0x80 != 0 {
		t.Errorf("expected busy bit cleared for data[3], got 0x%02x", km.data[3])
	}
	// 值应为 0（UNKNOWN，不是 old_key 也不是 new_key）
	if km.data[1] != 0 {
		t.Errorf("expected data[1]=0 (UNKNOWN), got %d", km.data[1])
	}
	if km.data[3] != 0 {
		t.Errorf("expected data[3]=0 (UNKNOWN), got %d", km.data[3])
	}
}
