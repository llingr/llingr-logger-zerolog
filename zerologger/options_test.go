// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-logger-zerolog Authors
// SPDX-License-Identifier: Apache-2.0

package zerologger_test

import (
	"bytes"
	"context"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/llingr/llingr-logger-zerolog/zerologger"
	"github.com/rs/zerolog"
)

// TestDefaultAddsCaller confirms the default (no options) New enables caller
// capture even when the host logger did not.
func TestDefaultAddsCaller(t *testing.T) {
	log, buf := newBuf(zerolog.DebugLevel) // host: no caller

	log.Info(context.Background(), "x")

	if _, ok := lastEntry(t, buf)["caller"]; !ok {
		t.Error("no caller field, want one (New should add caller by default)")
	}
}

// TestPreserveHostCallerRespectsCallerOff confirms PreserveHostCaller leaves a
// host that did NOT enable caller with caller still off: the adapter does not
// force it on. This is the escape hatch for hosts that disabled caller on purpose
// (cost or privacy).
func TestPreserveHostCallerRespectsCallerOff(t *testing.T) {
	var buf bytes.Buffer
	z := zerolog.New(&buf).Level(zerolog.DebugLevel) // host: no caller
	log := zerologger.New(z, zerologger.PreserveHostCaller())

	log.Info(context.Background(), "caller stays off")

	if c, ok := lastEntry(t, &buf)["caller"]; ok {
		t.Errorf("caller = %v, want none (PreserveHostCaller forced it on)", c)
	}
}

// TestPreserveHostCallerKeepsHostCaller confirms that when the host enabled caller
// itself (with the wrapper-aware skip count, as documented), PreserveHostCaller
// leaves it in place and the recorded site is the application's call site.
func TestPreserveHostCallerKeepsHostCaller(t *testing.T) {
	var buf bytes.Buffer
	// Host owns caller and accounts for this wrapper's one frame.
	z := zerolog.New(&buf).Level(zerolog.DebugLevel).
		With().CallerWithSkipFrameCount(zerolog.CallerSkipFrameCount + 1).Logger()
	log := zerologger.New(z, zerologger.PreserveHostCaller())

	_, _, line, _ := runtime.Caller(0)
	log.Info(context.Background(), "host caller") // must stay directly below the line above
	wantLine := line + 1

	caller, _ := lastEntry(t, &buf)["caller"].(string)
	if !strings.HasSuffix(caller, "options_test.go:"+strconv.Itoa(wantLine)) {
		t.Errorf("caller = %q, want options_test.go:%d (host caller + wrapper skip)", caller, wantLine)
	}
}
