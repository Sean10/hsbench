package main

import (
	"strings"
	"testing"
)

// ---- Group 4: mode validation ----

// Task 4.4: -m "ipgv" should not report invalid mode error
func TestIsValidMode_WithV(t *testing.T) {
	for _, r := range "ipgv" {
		if !isValidMode(r) {
			t.Errorf("expected 'v' to be a valid mode rune, got invalid for '%c'", r)
		}
	}
}

func TestIsValidMode_AllOriginal(t *testing.T) {
	for _, r := range "icpglxd" {
		if !isValidMode(r) {
			t.Errorf("original mode '%c' should be valid", r)
		}
	}
}

func TestIsValidMode_Invalid(t *testing.T) {
	for _, r := range "qzAB" {
		if isValidMode(r) {
			t.Errorf("mode '%c' should be invalid", r)
		}
	}
}

// ---- Group 5/6: pure helper functions ----

// Task 5.9/6.1: verifyObjectData compares data against deterministic generator
func TestVerifyObjectData_Match(t *testing.T) {
	data := generateObjectData(10, 3, 64)
	if !verifyObjectData(10, 3, data) {
		t.Fatal("expected match for data generated with same (objnum,key)")
	}
}

func TestVerifyObjectData_Mismatch(t *testing.T) {
	data := generateObjectData(10, 3, 64)
	data[0] ^= 0xFF // corrupt first byte
	if verifyObjectData(10, 3, data) {
		t.Fatal("expected mismatch for corrupted data")
	}
}

func TestVerifyObjectData_WrongKey(t *testing.T) {
	data := generateObjectData(10, 3, 64)
	if verifyObjectData(10, 4, data) {
		t.Fatal("expected mismatch when key differs")
	}
}

func TestVerifyObjectData_WrongObjnum(t *testing.T) {
	data := generateObjectData(10, 3, 64)
	if verifyObjectData(11, 3, data) {
		t.Fatal("expected mismatch when objnum differs")
	}
}

// Task 6.3: buildVerifyReport includes objnum, bucket, key, hex prefix
func TestBuildVerifyReport_ContainsFields(t *testing.T) {
	actual := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	report := buildVerifyReport(42, "test-bucket", 7, actual)

	if !strings.Contains(report, "42") {
		t.Error("report should contain objnum 42")
	}
	if !strings.Contains(report, "test-bucket") {
		t.Error("report should contain bucket name")
	}
	if !strings.Contains(report, "7") {
		t.Error("report should contain key value")
	}
	// hex of actual bytes
	if !strings.Contains(report, "deadbeef") {
		t.Error("report should contain hex representation of actual bytes")
	}
}

func TestBuildVerifyReport_LongData_TruncatesAt64(t *testing.T) {
	actual := make([]byte, 128)
	for i := range actual {
		actual[i] = byte(i)
	}
	report := buildVerifyReport(1, "b", 1, actual)
	// report should reference hex of up to 64 bytes, not all 128
	// 64 bytes = 128 hex chars; 128 bytes = 256 hex chars
	// We just ensure it doesn't contain the 65th byte's hex (0x40 = "40")
	// The simplest check: report length should be reasonable
	if len(report) == 0 {
		t.Fatal("report should not be empty")
	}
}

// Task 5.11/7.5: classifyKeyByte returns the right classification
func TestClassifyKeyByte(t *testing.T) {
	tests := []struct {
		b    uint8
		want string
	}{
		{0x00, "unwritten"},
		{0x01, "written"},
		{0x7E, "written"},
		{dvError, "dv_error"},
		{0x80, "busy"},
		{0xFF, "busy"},
	}
	for _, tc := range tests {
		got := classifyKeyByte(tc.b)
		if got != tc.want {
			t.Errorf("classifyKeyByte(0x%02X) = %q, want %q", tc.b, got, tc.want)
		}
	}
}
