// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-logger-zerolog Authors
// SPDX-License-Identifier: Apache-2.0

// Package zerologger wires an existing github.com/rs/zerolog logger into a
// nexus.Logger. It does not create loggers: the host supplies its own via New.
//
//   - Arguments: the message is logged verbatim (Msg, never the formatting Msgf);
//     the variadic args become structured fields as loose key/value pairs. See
//     normalize for the "!BADKEY" handling of a dangling or non-string key.
//   - Call site: by default New adds zerolog's caller field and skips this
//     package's own frame, so the recorded file:line is the application's, not
//     this package's. Pass PreserveHostCaller to leave the host's caller setting
//     untouched instead. Do not also enable caller on the host logger, or it will
//     appear twice.
//   - Context: by default the context is ignored. WithContextExtractor lazily
//     adds fields pulled from it (trace IDs and the like) on every line.
//
// zerolog has no buffering/flush step and no sugared variant, so unlike the zap
// binding this package has no Sync method and a single New constructor.
package zerologger

import (
	"context"

	"github.com/llingr/llingr-nexus/nexus"
	"github.com/rs/zerolog"
)

// callerSkip is the number of stack frames this package's wrapper method adds
// between the application's call site and zerolog's caller capture. It is added
// to zerolog's global CallerSkipFrameCount so the recorded caller is the
// application's, not ours. Each level method calls zerolog's Event.Msg directly,
// so exactly one frame needs skipping.
const callerSkip = 1

// Logger adapts a zerolog.Logger to nexus.Logger.
type Logger struct {
	z                  zerolog.Logger
	extract            ContextExtractor
	preserveHostCaller bool // construction-only: leave the host's caller config untouched
}

// compile-time assertion that Logger satisfies the nexus contract.
var _ nexus.Logger = (*Logger)(nil)

// New wraps an existing zerolog.Logger as a nexus.Logger. By default a caller
// field is added with this package's frame skipped, so the recorded caller is the
// application's call site, not this wrapper. The host does not need to enable
// caller itself; if it does, the field appears twice, so let the adapter own it
// (or pass PreserveHostCaller). Pass PreserveHostCaller to leave the host's caller
// configuration untouched.
func New(z zerolog.Logger, opts ...Option) *Logger {
	l := &Logger{}
	for _, opt := range opts {
		opt(l)
	}
	if !l.preserveHostCaller {
		z = z.With().CallerWithSkipFrameCount(zerolog.CallerSkipFrameCount + callerSkip).Logger()
	}
	l.z = z
	return l
}

// With returns a child Logger with the given fields bound to every subsequent log
// line. The arguments follow the same rules as the log methods (loose key/value
// pairs, with "!BADKEY" for a dangling or non-string key). Any ContextExtractor
// is inherited, and call-site transparency is preserved.
func (l *Logger) With(args ...any) *Logger {
	// Fields(nil) is a no-op in zerolog, so no guard is needed for the no-arg case.
	return &Logger{
		z:       l.z.With().Fields(normalize(args)).Logger(),
		extract: l.extract,
	}
}

// WithContextExtractor returns a child Logger that, for every subsequent call,
// lazily adds the fields the extractor pulls from the call's context.Context.
// Pass nil to disable extraction on the child. See ContextExtractor.
func (l *Logger) WithContextExtractor(extract ContextExtractor) *Logger {
	return &Logger{
		z:       l.z,
		extract: extract,
	}
}

// Error logs at error level. See the package documentation for argument
// semantics. The context is read only when a ContextExtractor is configured.
func (l *Logger) Error(ctx context.Context, msg string, args ...any) {
	e := l.z.Error()
	l.attach(e, ctx, args)
	e.Msg(msg)
}

// Warn logs at warn level. See Error for argument semantics.
func (l *Logger) Warn(ctx context.Context, msg string, args ...any) {
	e := l.z.Warn()
	l.attach(e, ctx, args)
	e.Msg(msg)
}

// Info logs at info level. See Error for argument semantics.
func (l *Logger) Info(ctx context.Context, msg string, args ...any) {
	e := l.z.Info()
	l.attach(e, ctx, args)
	e.Msg(msg)
}

// Debug logs at debug level. See Error for argument semantics.
func (l *Logger) Debug(ctx context.Context, msg string, args ...any) {
	e := l.z.Debug()
	l.attach(e, ctx, args)
	e.Msg(msg)
}

// attach adds the context extractor's fields and the call's structured fields to
// the event. It gates on Enabled, so a dropped line costs no extraction or field
// work, and it is a separate statement from the Msg call, so it adds no stack
// frame to zerolog's caller capture.
func (l *Logger) attach(e *zerolog.Event, ctx context.Context, args []any) {
	if !e.Enabled() {
		return
	}
	if l.extract != nil && ctx != nil {
		l.extract(ctx, e)
	}
	// Fields(nil) is a no-op in zerolog, so no guard is needed for the no-arg case.
	e.Fields(normalize(args))
}
