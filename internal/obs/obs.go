// Package obs holds logging setup and every telemetry attribute name as a
// constant.
//
// The constants are not ceremony. The gen_ai.* semantic conventions have no
// stable release and have already renamed gen_ai.system to
// gen_ai.provider.name; a spec bump must be a one-file diff rather than a
// repository-wide search.
package obs

import (
	"io"
	"log/slog"
	"strings"
)

// Attribute names. cred.* is CRED's own namespace.
const (
	AttrClaimID     = "cred.claim.id"
	AttrClaimCount  = "cred.claim.count"
	AttrEvidenceID  = "cred.evidence.id"
	AttrPrincipalID = "cred.principal.id"
	AttrRepo        = "cred.source.repo"
	AttrPath        = "cred.source.path"
	AttrLineStart   = "cred.source.line_start"
	AttrLineEnd     = "cred.source.line_end"

	//nolint:gosec // G101: an attribute name containing "token" is a token
	// count, not a credential.
	AttrRecallQueryTokens = "cred.recall.query_tokens"
	AttrRecallCandidates  = "cred.recall.candidates"
	AttrRecallAuthorized  = "cred.recall.authorized"
	AttrRecallOmitted     = "cred.recall.omitted"
	AttrRecallDurationMS  = "cred.recall.duration_ms"
	AttrRecallArm         = "cred.recall.arm"

	AttrEmbeddingModel = "cred.embedding.model_id"
	AttrSchemaVersion  = "cred.schema.version"

	// Usage and limits (PRD section 8). Scope and usage-kind are CRED's own
	// namespace; token counts use the gen_ai.* semantic conventions, which is
	// why those names live here — the conventions have no stable release and
	// have already renamed fields, so a spec bump is a one-file diff.
	AttrScopeKind    = "cred.scope.kind"
	AttrScopeValue   = "cred.scope.value"
	AttrUsageKind    = "cred.usage.kind"
	AttrDeniedReason = "cred.usage.denied_reason"
	AttrQuotaState   = "cred.usage.remaining"
	AttrScopeLive    = "cred.scope.live_claims"
	AttrPrunedCount  = "cred.usage.pruned"

	AttrInferenceCalls = "gen_ai.usage.calls"
	//nolint:gosec // G101: a gen_ai token-count attribute name, not a credential.
	AttrInputTokens = "gen_ai.usage.input_tokens"
	//nolint:gosec // G101: a gen_ai token-count attribute name, not a credential.
	AttrOutputTokens = "gen_ai.usage.output_tokens"
	AttrWallMS       = "cred.usage.wall_ms"

	// Correlation fields, hex-encoded W3C, snake_case. That format is what
	// Tempo, Loki, Jaeger and Honeycomb auto-link on.
	AttrTraceID = "trace_id"
	AttrSpanID  = "span_id"
)

// Never log claim or evidence body text. Ingested content is untrusted (L8)
// and logs are read by tools that were not designed to treat it as such. There
// is deliberately no attribute constant for claim statement or evidence text:
// the missing constant is the guard rail.

// NewLogger builds the process logger.
//
// Plain JSON or text to the writer, via log/slog, and deliberately not the
// OpenTelemetry logs bridge: an in-process logs SDK with a batching exporter
// is a mechanism that can silently drop records, and never silently dropping
// data is the product's core promise.
func NewLogger(w io.Writer, level, format string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(level)}
	if strings.EqualFold(format, "json") {
		return slog.New(slog.NewJSONHandler(w, opts))
	}
	return slog.New(slog.NewTextHandler(w, opts))
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
