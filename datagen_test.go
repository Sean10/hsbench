package main

import (
	"bytes"
	"testing"
)

func TestGenerateObjectData(t *testing.T) {
	tests := []struct {
		name        string
		a, b        []byte
		shouldEqual bool
	}{
		{
			name:        "deterministic: same inputs produce same output",
			a:           generateObjectData(42, 7, 128),
			b:           generateObjectData(42, 7, 128),
			shouldEqual: true,
		},
		{
			name:        "different key: same objnum different key produces different output",
			a:           generateObjectData(1, 0, 64),
			b:           generateObjectData(1, 1, 64),
			shouldEqual: false,
		},
		{
			name:        "different objnum: same key different objnum produces different output",
			a:           generateObjectData(0, 5, 64),
			b:           generateObjectData(1, 5, 64),
			shouldEqual: false,
		},
		{
			name:        "zero key objnum=0 vs objnum=1: different output even when key=0",
			a:           generateObjectData(0, 0, 64),
			b:           generateObjectData(1, 0, 64),
			shouldEqual: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if bytes.Equal(tc.a, tc.b) != tc.shouldEqual {
				if tc.shouldEqual {
					t.Fatalf("expected equal outputs but got different data")
				} else {
					t.Fatalf("expected different outputs but got identical data")
				}
			}
		})
	}
}

// TestGenerateObjectData_SizeZero verifies that size=0 returns an empty
// (non-nil) slice without panicking.
func TestGenerateObjectData_SizeZero(t *testing.T) {
	result := generateObjectData(0, 0, 0)
	if result == nil {
		t.Fatal("expected non-nil slice for size=0, got nil")
	}
	if len(result) != 0 {
		t.Fatalf("expected length 0 for size=0, got %d", len(result))
	}
}
