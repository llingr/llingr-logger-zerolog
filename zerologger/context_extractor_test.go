// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-logger-zerolog Authors
// SPDX-License-Identifier: Apache-2.0

package zerologger_test

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
)

// ctxKey is a private context key for the extractor tests.
type ctxKey struct{}

// TestContextExtractorAddsFlatFields confirms a configured extractor pulls fields
// from the call's context and adds them flat into the entry, alongside explicit
// args.
func TestContextExtractorAddsFlatFields(t *testing.T) {
	log, buf := newBuf(zerolog.DebugLevel)
	log = log.WithContextExtractor(func(ctx context.Context, e *zerolog.Event) {
		if id, ok := ctx.Value(ctxKey{}).(string); ok {
			e.Str("trace_id", id)
		}
	})

	ctx := context.WithValue(context.Background(), ctxKey{}, "trace-abc")
	log.Info(ctx, "handled", "key", "k1")

	e := lastEntry(t, buf)
	if e["trace_id"] != "trace-abc" {
		t.Errorf("trace_id = %v, want trace-abc (extractor not applied)", e["trace_id"])
	}
	if e["key"] != "k1" {
		t.Errorf("explicit field key = %v, want k1", e["key"])
	}
}

// TestContextExtractorIsLazy proves the extractor does not run for an entry that
// is dropped by the level gate, and does run (once) on one that is emitted: the
// whole point of gating on Event.Enabled.
func TestContextExtractorIsLazy(t *testing.T) {
	log, _ := newBuf(zerolog.WarnLevel)
	calls := 0
	log = log.WithContextExtractor(func(_ context.Context, e *zerolog.Event) {
		calls++
		e.Str("trace_id", "x")
	})
	ctx := context.Background()

	log.Debug(ctx, "below threshold") // dropped: extractor must not run
	log.Info(ctx, "below threshold")  // dropped: extractor must not run
	if calls != 0 {
		t.Fatalf("extractor ran %d times for dropped lines, want 0", calls)
	}

	log.Error(ctx, "above threshold") // emitted: extractor runs once
	if calls != 1 {
		t.Errorf("extractor ran %d times, want 1", calls)
	}
}

// TestNilExtractorDisablesOnChild confirms passing nil to WithContextExtractor
// turns extraction off for the child even when the parent had one.
func TestNilExtractorDisablesOnChild(t *testing.T) {
	log, buf := newBuf(zerolog.DebugLevel)
	withExtractor := log.WithContextExtractor(func(_ context.Context, e *zerolog.Event) {
		e.Str("trace_id", "should-not-appear")
	})
	disabled := withExtractor.WithContextExtractor(nil)

	disabled.Info(context.Background(), "no extraction")

	if _, ok := lastEntry(t, buf)["trace_id"]; ok {
		t.Error("trace_id present, want none (nil extractor should disable it on the child)")
	}
}
