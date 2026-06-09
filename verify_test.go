package main

import (
	"strings"
	"testing"
)

// ---- Group 7: runVerify helper logic ----

// Task 7.1: v mode skips objects with key=0 (never written)
func TestRunVerify_SkipsUnwritten(t *testing.T) {
	// classifyKeyByte(0) == "unwritten" means it should be skipped
	class := classifyKeyByte(0)
	if class != "unwritten" {
		t.Fatalf("expected 'unwritten' for key=0, got %q", class)
	}
}

// Task 7.3: v mode skips busy objects and counts them as unknown
func TestRunVerify_SkipsBusy(t *testing.T) {
	class := classifyKeyByte(0x80) // busy flag set
	if class != "busy" {
		t.Fatalf("expected 'busy' for byte 0x80, got %q", class)
	}
	class = classifyKeyByte(0x83) // busy with key=3
	if class != "busy" {
		t.Fatalf("expected 'busy' for byte 0x83, got %q", class)
	}
}

// Task 7.5: v mode skips DV_ERROR (0x7F) objects and counts them as dv_error
func TestRunVerify_SkipsDVError(t *testing.T) {
	class := classifyKeyByte(dvError)
	if class != "dv_error" {
		t.Fatalf("expected 'dv_error' for dvError byte, got %q", class)
	}
}

// Task 7.7: verify summary report format
func TestRunVerify_SummaryFormat(t *testing.T) {
	// buildVerifySummary should include all counters
	summary := buildVerifySummary(100, 80, 75, 5, 10, 5, 0)
	checks := []string{"total=100", "verified=80", "pass=75", "fail=5", "unknown=10", "dv_error=5"}
	for _, expected := range checks {
		if !strings.Contains(summary, expected) {
			t.Errorf("summary missing %q, got: %s", expected, summary)
		}
	}
}
