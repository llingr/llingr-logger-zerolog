// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-logger-zerolog Authors
// SPDX-License-Identifier: Apache-2.0

package zerologger_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/llingr/llingr-logger-zerolog/zerologger"
	"github.com/llingr/llingr-nexus/nexus"
	"github.com/rs/zerolog"
)

// newBuf returns a Logger writing JSON to a buffer (caller enabled by default, as
// New does), plus the buffer so tests can inspect the emitted lines.
func newBuf(level zerolog.Level) (*zerologger.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	z := zerolog.New(&buf).Level(level)
	return zerologger.New(z), &buf
}

// entries parses every non-empty JSON line in buf.
func entries(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	sc := bufio.NewScanner(bytes.NewReader(buf.Bytes()))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("parse log line %q: %v", line, err)
		}
		out = append(out, m)
	}
	return out
}

// lastEntry parses the final JSON log line in buf.
func lastEntry(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	all := entries(t, buf)
	if len(all) == 0 {
		t.Fatalf("no log line emitted; buffer=%q", buf.String())
	}
	return all[len(all)-1]
}

// TestCallerIsApplicationNotPackage is the headline guarantee: the recorded
// caller must point at this test file (the invocation site), never at
// zerologger.go (the wrapper). The log call sits on the line immediately after
// runtime.Caller so the expected line number is exact; keep them adjacent.
func TestCallerIsApplicationNotPackage(t *testing.T) {
	log, buf := newBuf(zerolog.DebugLevel)

	_, _, line, _ := runtime.Caller(0)
	log.Info(context.Background(), "hello") // must stay directly below the line above
	wantLine := line + 1

	caller, ok := lastEntry(t, buf)["caller"].(string)
	if !ok || caller == "" {
		t.Fatal("no caller field: New should enable caller capture")
	}
	if strings.Contains(filepath.ToSlash(caller), "zerologger/zerologger.go") {
		t.Errorf("caller points at the wrapper package: %q", caller)
	}
	wantSuffix := "zerologger_test.go:" + strconv.Itoa(wantLine)
	if !strings.HasSuffix(caller, wantSuffix) {
		t.Errorf("caller = %q, want it to end with %q (wrapper frame not skipped)", caller, wantSuffix)
	}
}

// TestMessageNeverReformatted proves the logger does not run fmt.Sprintf: a
// percent verb in the message survives verbatim even with a trailing arg.
func TestMessageNeverReformatted(t *testing.T) {
	log, buf := newBuf(zerolog.DebugLevel)

	log.Info(context.Background(), "progress 50% complete", "n", 1)

	if msg, _ := lastEntry(t, buf)["message"].(string); msg != "progress 50% complete" {
		t.Errorf("message = %q, want it verbatim", msg)
	}
}

// TestLevelFiltering checks each level routes correctly and sub-threshold lines
// are dropped by zerolog.
func TestLevelFiltering(t *testing.T) {
	log, buf := newBuf(zerolog.WarnLevel)
	ctx := context.Background()

	log.Debug(ctx, "debug") // dropped
	log.Info(ctx, "info")   // dropped
	log.Warn(ctx, "warn")   // kept
	log.Error(ctx, "error") // kept

	got := entries(t, buf)
	if len(got) != 2 {
		t.Fatalf("want 2 entries above warn, got %d: %q", len(got), buf.String())
	}
	if got[0]["level"] != "warn" || got[1]["level"] != "error" {
		t.Errorf("levels = %v, %v; want warn, error", got[0]["level"], got[1]["level"])
	}
}

// TestWithBindsFieldsAndKeepsCaller verifies child loggers carry bound fields and
// still attribute the application call site.
func TestWithBindsFieldsAndKeepsCaller(t *testing.T) {
	log, buf := newBuf(zerolog.DebugLevel)

	child := log.With("topic", "payments")
	_, _, line, _ := runtime.Caller(0)
	child.Info(context.Background(), "bound") // must stay directly below the line above
	wantLine := line + 1

	e := lastEntry(t, buf)
	if e["topic"] != "payments" {
		t.Errorf("bound field topic = %v, want payments", e["topic"])
	}
	caller, _ := e["caller"].(string)
	if !strings.HasSuffix(caller, "zerologger_test.go:"+strconv.Itoa(wantLine)) {
		t.Errorf("child caller = %q, want zerologger_test.go:%d", caller, wantLine)
	}
}

// TestSatisfiesNexusLogger is a runtime companion to the compile-time assertion
// in the package.
func TestSatisfiesNexusLogger(t *testing.T) {
	var _ nexus.Logger = zerologger.New(zerolog.Nop())
}
