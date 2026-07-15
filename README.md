# llingr-logger-zerolog

[![Go Reference](https://pkg.go.dev/badge/github.com/llingr/llingr-logger-zerolog.svg)](https://pkg.go.dev/github.com/llingr/llingr-logger-zerolog)
[![Pipeline](https://github.com/llingr/llingr-logger-zerolog/actions/workflows/pipeline.yml/badge.svg)](https://github.com/llingr/llingr-logger-zerolog/actions/workflows/pipeline.yml)
[![Tag](https://img.shields.io/github/v/tag/llingr/llingr-logger-zerolog)](https://github.com/llingr/llingr-logger-zerolog/tags)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/llingr/llingr-logger-zerolog)](go.mod)

A [`nexus.Logger`](https://github.com/llingr/llingr-nexus) implementation
backed by [zerolog](https://github.com/rs/zerolog).

`llingr-nexus` ships a zero-dependency `DefaultLogger` (slog-based) for
convenience. For production deployments the recommendation is to plug in an
established logging library; this module is one such binding (zerolog). The
interface is small, so a zap, logrus, or bespoke binding is equally valid; this
is just the zerolog one.

- **Module:** `github.com/llingr/llingr-logger-zerolog`
- **Go:** 1.25+
- **Licence:** Apache-2.0
- **Depends on:** `llingr-nexus`, `github.com/rs/zerolog`

This binding targets Go 1.25, a little ahead of the rest of the llingr
ecosystem. That is intentional: the patched `golang.org/x/sys` needed to keep
its dependency tree free of known advisories requires Go 1.25.

## Scope

`nexus.Logger` is the small interface that a consumer built on
[llingr-nexus](https://github.com/llingr/llingr-nexus) uses for its own
operational logging: lifecycle events, processing errors, rebalances, and the
like. This package adapts that interface onto zerolog.

It is not intended to be your application's logger. The purpose is narrow: if
your service already uses zerolog, wiring this binding into a llingr-nexus
consumer sends the consumer's operational log lines through the zerolog logger
you already have (your configuration, your encoder, your sinks), so you run one
logging pipeline rather than two. If you do not use zerolog, there is no reason
to adopt it: llingr-nexus already ships a zero-dependency default logger.

Because `nexus.Logger` is deliberately minimal (four levels, a message, and flat
key/value fields), it does not expose zerolog's full API. In particular:

- per-call fields arrive as loose key/value pairs, not the fluent typed chain
  (`.Str().Int()...`);
- only Error, Warn, Info, and Debug are available (no Trace, Fatal, or Panic);
- the `.Err()` / `.Stack()` conventions, nested values (`.Dict`, `.Array`,
  `LogObjectMarshaler`), and the `zerolog.Ctx` logger-in-context pattern are not
  part of the interface;
- the message is written verbatim (`Msg`, never the formatting `Msgf`).

None of this limits your application: everywhere else, log through zerolog
directly with its complete API. This binding only governs how a consumer's
operational logging reaches your zerolog logger, where a message and a few
fields at four levels is all that is needed.

## Call-site transparency

The wrapper is **call-site transparent**: the file and line zerolog records
belong to the application code that called the log method, never to this
package. Each log method adds one stack frame between your code and zerolog's
caller capture, so by default the constructor enables caller capture and adds a
matching skip (`zerolog.CallerSkipFrameCount + 1`) to chop that frame off. You
get the call site of the invocation, not the call site of the logger.

You do **not** need to enable caller on the host logger; the constructor does
it. In fact you should **not** also call `.Caller()` on the host logger, because
zerolog would then emit the `caller` field twice (unlike zap, zerolog appends a
caller field each time it is requested).

If your host has made a deliberate caller decision the adapter should not
override (caller turned off for cost or privacy, or configured with its own
skip), pass `zerologger.PreserveHostCaller()`:

```go
log := zerologger.New(z, zerologger.PreserveHostCaller())
```

The adapter then leaves caller entirely to the host. Note that, unlike zap,
zerolog cannot adjust the skip without re-adding the caller field, so under
`PreserveHostCaller` the host owns the skip: if it enables caller itself it
should add one frame for this wrapper (`zerolog.CallerSkipFrameCount + 1`) for
the reported site to be the application's.

There is no `Sync` method: zerolog writes each entry straight to its writer with
no buffering to flush, so the zap binding's `Sync`/ENOTTY handling has no
zerolog equivalent.

## Usage

This package **wires an existing logger; it does not create one.** The host
application supplies its own zerolog logger, configured however it likes, and
you wrap it:

```go
import (
	"github.com/llingr/llingr-logger-zerolog/zerologger"
	"github.com/rs/zerolog"
)

z := app.Logger()                 // the host's own zerolog.Logger, their config
log := zerologger.New(z)          // wrap it as a nexus.Logger

consumer, _ := builder.
	WithLogger(log).
	Build()
```

There is a single `New` constructor: zerolog has no sugared/typed split (and no
`NewProduction`-style preset), so there is nothing to mirror from the zap
binding's `NewSugared`. Building a logger is zerolog's job, and forcing a
configuration is exactly what this package avoids.

## Arguments

The message is logged **verbatim**: this package calls zerolog's `Msg`, never
the formatting `Msgf`, so a literal `%` is safe. The variadic arguments become
structured fields as **loose key/value pairs** (slog style):

```go
log.Info(ctx, "processed", "key", msg.Key, "partition", msg.Partition)
```

Unlike the zap binding there is no separate "native field" path, because zerolog
has no standalone field value type; fields are written onto the event. The pairs
are still written with zerolog's typed, allocation-light dispatch.

A trailing key with no value, or a non-string where a key is expected, is
recorded under `!BADKEY`, matching slog and zap's `SugaredLogger`. This is
deliberate: zerolog's own `Fields` silently *drops* malformed input (it
truncates an odd-length list and skips any pair with a non-string key), so the
adapter normalizes the arguments first to keep that data instead of losing it.

The `context.Context` argument is accepted to satisfy the interface; this
package does not read from it unless a `ContextExtractor` is configured.

`With` returns a child logger with fields bound to every subsequent line,
preserving call-site transparency:

```go
topicLog := log.With("topic", "payments")
topicLog.Warn(ctx, "retry scheduled", "attempt", 2)
```

## Context fields

`nexus.Logger` threads a `context.Context` into every call, and most
implementations discard it. Register a `ContextExtractor` to pull ambient
correlation data (trace IDs, request IDs, tenant, ...) out of that context and
add it to every line. The extractor is handed the live `*zerolog.Event` and adds
fields directly:

```go
log = log.WithContextExtractor(func(ctx context.Context, e *zerolog.Event) {
	if id, ok := ctx.Value(traceKey).(string); ok {
		e.Str("trace_id", id)
	}
})

log.Info(ctx, "handled") // => {... "trace_id":"abc123"}
```

This signature differs from the zap binding (which returns a `[]zap.Field`)
because zerolog has no field value type: the idiomatic, allocation-light way to
add fields is to call the event's typed methods. So:

- **Lazy.** The extractor runs only when an entry clears the level gate (the
  adapter checks `Event.Enabled` first). A suppressed `Debug` line costs no
  extraction.
- **Flat.** The fields land at the top level of the entry, not nested under a
  key.

Which keys to read is application-specific, which is why it is a plugged-in
function rather than built in. Pass `nil` to disable it on a child logger.

## API

| Constructor                                | Use                             |
|--------------------------------------------|---------------------------------|
| `New(zerolog.Logger, ...Option) *Logger`   | Wrap an existing zerolog logger |

| Option                 | Effect                                                                            |
|------------------------|-----------------------------------------------------------------------------------|
| `PreserveHostCaller()` | Do not add a caller field; keep the host's caller setting (it then owns the skip) |

Methods: `Error`, `Warn`, `Info`, `Debug` (the `nexus.Logger` interface), plus
`With(args ...any) *Logger` and
`WithContextExtractor(ContextExtractor) *Logger`.
