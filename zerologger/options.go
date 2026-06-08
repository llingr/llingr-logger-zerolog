// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-logger-zerolog Authors
// SPDX-License-Identifier: Apache-2.0

package zerologger

// Option configures a Logger at construction time, passed to New.
// See PreserveHostCaller.
type Option func(*Logger)

// PreserveHostCaller leaves the host logger's caller configuration untouched: the
// constructor will not add a caller field. By default New adds one (with this
// package's wrapper frame skipped) so the recorded site is the application's call
// site. Use this when the host has made a deliberate caller decision the adapter
// should not override, for example caller turned off for cost or privacy, or
// caller configured by the host with its own skip count.
//
// Note: unlike zap, zerolog cannot adjust the caller skip without (re-)adding the
// caller field, so PreserveHostCaller hands caller entirely to the host. If the
// host enables caller itself, it owns the skip count and should add one frame for
// this wrapper (zerolog.CallerSkipFrameCount + 1) for the reported site to be the
// application's rather than this package's.
func PreserveHostCaller() Option {
	return func(l *Logger) {
		l.preserveHostCaller = true
	}
}
