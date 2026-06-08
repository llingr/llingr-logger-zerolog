// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-logger-zerolog Authors
// SPDX-License-Identifier: Apache-2.0

package zerologger

// badKey labels a loose argument that could not be paired into a key/value, in
// line with slog and zap's SugaredLogger.
const badKey = "!BADKEY"

// normalize turns the nexus.Logger variadic into a clean, even-length slice of
// alternating string-key / value pairs, ready for zerolog's Event.Fields and
// Context.Fields. A trailing key with no value, or a non-string where a key is
// expected, is recorded under "!BADKEY", matching slog and zap's SugaredLogger.
//
// This step is deliberate. zerolog's own Fields silently DROPS malformed input:
// it truncates an odd-length slice and skips any pair whose key is not a string.
// Passing raw args straight through would therefore lose data. Normalizing first
// keeps the loss-free "!BADKEY" semantics, and zerolog's Fields then writes the
// well-formed pairs with its typed, allocation-light dispatch. An empty argument
// list returns nil, so the common no-field call is free.
func normalize(args []any) []any {
	if len(args) == 0 {
		return nil
	}
	out := make([]any, 0, len(args))
	for i := 0; i < len(args); {
		if i == len(args)-1 {
			out = append(out, badKey, args[i])
			break
		}
		key, ok := args[i].(string)
		if !ok {
			out = append(out, badKey, args[i])
			i++
			continue
		}
		out = append(out, key, args[i+1])
		i += 2
	}
	return out
}
