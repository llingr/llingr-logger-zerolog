// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-logger-zerolog Authors
// SPDX-License-Identifier: Apache-2.0

package zerologger_test

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
)

// TestStructuredFieldsForwarded confirms the variadic args are treated as
// key/value pairs (slog semantics), matching nexus.DefaultLogger. Numbers come
// back as float64 because the assertions parse the JSON output.
func TestStructuredFieldsForwarded(t *testing.T) {
	log, buf := newBuf(zerolog.DebugLevel)

	log.Info(context.Background(), "processed", "key", "b7K2mX9qRf4WpA", "partition", 3)

	e := lastEntry(t, buf)
	if e["message"] != "processed" {
		t.Errorf("message = %v, want processed", e["message"])
	}
	if e["level"] != "info" {
		t.Errorf("level = %v, want info", e["level"])
	}
	if e["key"] != "b7K2mX9qRf4WpA" {
		t.Errorf("field key = %v, want b7K2mX9qRf4WpA", e["key"])
	}
	if e["partition"] != float64(3) {
		t.Errorf("field partition = %v (%T), want 3", e["partition"], e["partition"])
	}
}

// TestNonStringKeyMidListAdvancesByOne covers a non-string in a key position that
// is NOT the final argument: it must land under !BADKEY, and the key/value pair
// that follows must still parse correctly. That proves the scan advanced by
// exactly one past the bad key. Without this case, the advance in that branch is
// untested (a gap mutation testing surfaces: a flipped increment there goes
// undetected by coverage alone).
func TestNonStringKeyMidListAdvancesByOne(t *testing.T) {
	log, buf := newBuf(zerolog.DebugLevel)

	log.Info(context.Background(), "mixed", 42, "topic", "payments")

	e := lastEntry(t, buf)
	if e["!BADKEY"] != float64(42) {
		t.Errorf("field !BADKEY = %v (%T), want 42", e["!BADKEY"], e["!BADKEY"])
	}
	if e["topic"] != "payments" {
		t.Errorf("field topic = %v, want payments (scan did not advance past the bad key correctly)", e["topic"])
	}
}

// TestDanglingKeyBadKey covers a trailing value with no following pair: it must
// land under !BADKEY (matching slog), where zerolog's own Fields would silently
// drop it.
func TestDanglingKeyBadKey(t *testing.T) {
	log, buf := newBuf(zerolog.DebugLevel)

	log.Error(context.Background(), "drain failed", "topic", "payments", "dangling")

	e := lastEntry(t, buf)
	if e["topic"] != "payments" {
		t.Errorf("field topic = %v, want payments", e["topic"])
	}
	if e["!BADKEY"] != "dangling" {
		t.Errorf("field !BADKEY = %v, want dangling", e["!BADKEY"])
	}
}
