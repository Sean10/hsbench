package main

import (
	"os"
	"testing"
)

// Task 2.1: NewKeyMap 创建新文件时初始化为全零
func TestNewKeyMap_CreatesNewFileAllZeros(t *testing.T) {
	path := t.TempDir() + "/keymap.dat"
	const objectCount int64 = 10

	km, err := NewKeyMap(path, objectCount)
	if err != nil {
		t.Fatalf("NewKeyMap failed: %v", err)
	}

	if int64(len(km.data)) != objectCount {
		t.Fatalf("expected data length %d, got %d", objectCount, len(km.data))
	}
	for i, b := range km.data {
		if b != 0 {
			t.Fatalf("expected data[%d]=0, got %d", i, b)
		}
	}
}

// Task 2.3: NewKeyMap 加载已有文件保留现有 key 值
func TestNewKeyMap_LoadsExistingFile(t *testing.T) {
	path := t.TempDir() + "/keymap.dat"
	const objectCount int64 = 5

	// 创建文件并写入已知数据
	existing := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	if err := os.WriteFile(path, existing, 0644); err != nil {
		t.Fatalf("setup WriteFile failed: %v", err)
	}

	km, err := NewKeyMap(path, objectCount)
	if err != nil {
		t.Fatalf("NewKeyMap failed: %v", err)
	}

	if int64(len(km.data)) != objectCount {
		t.Fatalf("expected data length %d, got %d", objectCount, len(km.data))
	}
	for i, b := range km.data {
		if b != existing[i] {
			t.Fatalf("expected data[%d]=%d, got %d", i, existing[i], b)
		}
	}
}

// Task 2.5: acquireBusy 设置 bit7 并返回 old_key
func TestAcquireBusy_SetsBit7AndReturnsOldKey(t *testing.T) {
	path := t.TempDir() + "/keymap.dat"
	const objectCount int64 = 3

	km, err := NewKeyMap(path, objectCount)
	if err != nil {
		t.Fatalf("NewKeyMap failed: %v", err)
	}

	// 手动设置已知值
	km.data[1] = 0x05

	oldKey, ok := km.acquireBusy(1)
	if !ok {
		t.Fatal("expected acquireBusy to succeed")
	}
	if oldKey != 0x05 {
		t.Fatalf("expected oldKey=5, got %d", oldKey)
	}
	// bit7 应已置位
	if km.data[1] != (0x05 | 0x80) {
		t.Fatalf("expected data[1]=0x85, got 0x%02x", km.data[1])
	}
}

// Task 2.5 补充: acquireBusy 已 busy 时返回 false
func TestAcquireBusy_AlreadyBusy_ReturnsFalse(t *testing.T) {
	path := t.TempDir() + "/keymap.dat"
	km, err := NewKeyMap(path, 3)
	if err != nil {
		t.Fatalf("NewKeyMap failed: %v", err)
	}

	km.data[0] = 0x85 // 已 busy

	_, ok := km.acquireBusy(0)
	if ok {
		t.Fatal("expected acquireBusy to fail on already-busy byte")
	}
}

// Task 2.7: releaseBusy 成功时写入 new_key（清除 bit7）
func TestReleaseBusy_SuccessWritesNewKey(t *testing.T) {
	path := t.TempDir() + "/keymap.dat"
	km, err := NewKeyMap(path, 3)
	if err != nil {
		t.Fatalf("NewKeyMap failed: %v", err)
	}

	km.data[2] = 0x85 // busy 状态
	km.releaseBusy(2, 0x06, true)

	if km.data[2] != 0x06 {
		t.Fatalf("expected data[2]=0x06, got 0x%02x", km.data[2])
	}
}

// Task 2.9: releaseBusy 失败时恢复 old_key
func TestReleaseBusy_FailureRestoresOldKey(t *testing.T) {
	path := t.TempDir() + "/keymap.dat"
	km, err := NewKeyMap(path, 3)
	if err != nil {
		t.Fatalf("NewKeyMap failed: %v", err)
	}

	km.data[0] = 0x83 // busy with old_key=3
	km.releaseBusy(0, 0x03, false)

	if km.data[0] != 0x03 {
		t.Fatalf("expected data[0]=0x03 (restored), got 0x%02x", km.data[0])
	}
}

// Task 2.11: key 递增循环（126 → 1）
func TestNextKey_Wraps126To1(t *testing.T) {
	tests := []struct {
		oldKey   uint8
		expected uint8
	}{
		{0, 1},   // 0 % 126 + 1 = 1
		{1, 2},   // 1 % 126 + 1 = 2
		{125, 126}, // 125 % 126 + 1 = 126
		{126, 1}, // 126 % 126 + 1 = 1  (wrap)
		{5, 6},
	}

	for _, tc := range tests {
		got := NextKey(tc.oldKey)
		if got != tc.expected {
			t.Errorf("NextKey(%d) = %d, want %d", tc.oldKey, got, tc.expected)
		}
	}
}

// Task 2.13: Sync 将内存数据写回文件
func TestSync_WritesDataToFile(t *testing.T) {
	path := t.TempDir() + "/keymap.dat"
	km, err := NewKeyMap(path, 4)
	if err != nil {
		t.Fatalf("NewKeyMap failed: %v", err)
	}

	km.data[0] = 0x01
	km.data[1] = 0x02
	km.data[2] = 0x03
	km.data[3] = 0x04

	if err := km.Sync(); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	want := []byte{0x01, 0x02, 0x03, 0x04}
	for i, b := range want {
		if got[i] != b {
			t.Fatalf("file[%d]=%d, want %d", i, got[i], b)
		}
	}
}

// HasBusy 测试
func TestHasBusy(t *testing.T) {
	path := t.TempDir() + "/keymap.dat"
	km, err := NewKeyMap(path, 4)
	if err != nil {
		t.Fatalf("NewKeyMap failed: %v", err)
	}

	if km.HasBusy() {
		t.Fatal("expected HasBusy=false on fresh keymap")
	}

	km.data[2] = 0x85
	if !km.HasBusy() {
		t.Fatal("expected HasBusy=true after setting busy bit")
	}
}
