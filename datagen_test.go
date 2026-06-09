package main

import (
	"testing"
)

// Task 1.1: 相同输入返回相同输出（确定性）
func TestGenerateObjectData_Deterministic(t *testing.T) {
	a := generateObjectData(42, 7, 128)
	b := generateObjectData(42, 7, 128)
	if len(a) != 128 || len(b) != 128 {
		t.Fatalf("expected length 128, got %d and %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("byte %d differs: %d vs %d (not deterministic)", i, a[i], b[i])
		}
	}
}

// Task 1.3: 不同 key 返回不同数据
func TestGenerateObjectData_DifferentKey(t *testing.T) {
	a := generateObjectData(1, 0, 64)
	b := generateObjectData(1, 1, 64)
	same := true
	for i := range a {
		if a[i] != b[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("expected different output for different key, got identical bytes")
	}
}

// Task 1.5: 不同 objnum 返回不同数据
func TestGenerateObjectData_DifferentObjnum(t *testing.T) {
	a := generateObjectData(0, 5, 64)
	b := generateObjectData(1, 5, 64)
	same := true
	for i := range a {
		if a[i] != b[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("expected different output for different objnum, got identical bytes")
	}
}
