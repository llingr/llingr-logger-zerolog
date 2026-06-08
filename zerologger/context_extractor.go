// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-logger-zerolog Authors
// SPDX-License-Identifier: Apache-2.0

package zerologger

import (
	"context"

	"github.com/rs/zerolog"
)

// ContextExtractor pulls ambient correlation data (trace IDs, request IDs,
// tenant, and so on) out of a context.Context and adds it to a log entry. It is
// handed the live *zerolog.Event and adds fields directly with zerolog's typed
// methods (e.Str, e.Int, ...), which is the allocation-light, idiomatic way.
//
// This signature differs from the zap binding, whose extractor returns a
// []zap.Field, because zerolog has no standalone field value type: fields are
// written onto the event. The event may be nil/disabled, but zerolog's field
// methods are nil-safe, and the adapter only invokes the extractor when the entry
// is enabled.
//
// It is invoked lazily, only when an entry clears the level gate, so extraction
// costs nothing for dropped lines. The fields land flat at the top level of the
// entry. Register one with Logger.WithContextExtractor.
type ContextExtractor func(ctx context.Context, e *zerolog.Event)
