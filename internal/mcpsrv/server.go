// Package mcpsrv exposes CRED over the Model Context Protocol.
//
// One tool, read-only. Against a category median of nine tools and two to five
// required configuration decisions, one read-only tool needing neither an API
// key nor a provider choice is the differentiated position — and it is only
// available before a write path exists.
//
// Nothing here is persisted keyed on MCP session identity. Sessions and the
// initialize handshake are being removed from the specification, so anything
// keyed on them is state with a scheduled demolition date.
package mcpsrv

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/obs"
	"github.com/canhta/cred/internal/recall"
)

// Version is reported to clients.
//
// While CRED is 0.x, minor bumps may change the MCP tool schema. The tool
// surface is the public API — not the Go package — and 1.0 arrives when it
// stops moving.
const Version = "0.1.0"

// RecallInput is the tool's argument schema.
type RecallInput struct {
	Query string `json:"query" jsonschema:"what the agent is trying to find out"`
	Limit int    `json:"limit,omitempty" jsonschema:"maximum claims to return, default 5"`
}

// RecallEvidence is one evidence pointer in the response.
type RecallEvidence struct {
	Path      string `json:"path"`
	LineStart int    `json:"lineStart"`
	LineEnd   int    `json:"lineEnd"`
	Text      string `json:"text"`
}

// RecallClaim is one returned claim.
type RecallClaim struct {
	ID         string           `json:"id"`
	Kind       string           `json:"kind"`
	Statement  string           `json:"statement"`
	Score      float64          `json:"score"`
	Confidence float64          `json:"confidence"`
	Evidence   []RecallEvidence `json:"evidence"`
}

// RecallOutput is the assembled package.
type RecallOutput struct {
	Claims []RecallClaim `json:"claims"`

	// AsOf and StalenessSeconds are on every response. Truncation is reported
	// explicitly with a count, because a silent truncation is a lie.
	AsOf             string  `json:"asOf"`
	StalenessSeconds float64 `json:"stalenessSeconds"`
	Returned         int     `json:"returned"`
	Omitted          int     `json:"omitted"`
	TokensUsed       int     `json:"tokensUsed"`
	TokenBudget      int     `json:"tokenBudget"`
}

// Attestor writes a human-attested claim. It is the write half of the MCP
// surface: an agent calling `remember` is asserting a claim, which L1 counts as
// evidence. It is deliberately narrow — the server holds no nominator and no
// automatic-write path, so the tool cannot be tricked into extraction. The
// automatic path runs off the turn, driven by a hook, not by this server.
type Attestor interface {
	Attest(ctx context.Context, statement, kind string, principal claim.PrincipalID) (string, error)
}

// Server wires the recall service to an MCP server, plus an optional write path.
type Server struct {
	recall    *recall.Service
	attest    Attestor
	principal claim.PrincipalID
	log       *slog.Logger
}

// New builds a Server. attest may be nil, in which case the server is read-only
// and registers no remember tool — the read path's zero-config guarantee, kept
// available for a deployment that wants recall alone.
func New(svc *recall.Service, attest Attestor, principal claim.PrincipalID, log *slog.Logger) *Server {
	return &Server{recall: svc, attest: attest, principal: principal, log: log}
}

// ServeStdio runs the server over stdio until the client disconnects.
func (s *Server) ServeStdio(ctx context.Context) error {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "cred",
		Version: Version,
	}, nil)

	mcp.AddTool(srv, &mcp.Tool{
		Name: "recall",
		Description: "Retrieve what this organization already knows about a topic, " +
			"with the evidence each claim rests on. Read-only: it stores nothing " +
			"and changes nothing.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
			Title:        "Recall organizational memory",
		},
	}, s.handleRecall)

	if s.attest != nil {
		mcp.AddTool(srv, &mcp.Tool{
			Name: "remember",
			Description: "Contribute a claim to organizational memory. Use this " +
				"when the user states a decision, convention, constraint, or " +
				"lesson worth keeping. The statement you assert becomes its own " +
				"evidence, attributed to you. It is visible via `cred log` and " +
				"reversible via `cred forget`; it is never silent or permanent.",
			Annotations: &mcp.ToolAnnotations{
				Title: "Remember a claim",
			},
		}, s.handleRemember)
	}

	return srv.Run(ctx, &mcp.StdioTransport{})
}

// RememberInput is the write tool's argument schema.
type RememberInput struct {
	Statement string `json:"statement" jsonschema:"the claim to remember, kept short"`
	Kind      string `json:"kind,omitempty" jsonschema:"one of Convention, Decision, Constraint, RejectedApproach, Failure, Reference"`
}

// RememberOutput confirms what was written, so the write is visible in-band.
type RememberOutput struct {
	ID        string `json:"id"`
	Statement string `json:"statement"`
	Stored    bool   `json:"stored"`
}

func (s *Server) handleRemember(ctx context.Context, _ *mcp.CallToolRequest, in RememberInput) (
	*mcp.CallToolResult, RememberOutput, error,
) {
	statement := strings.TrimSpace(in.Statement)
	if statement == "" {
		return nil, RememberOutput{}, fmt.Errorf("statement must not be empty")
	}

	id, err := s.attest.Attest(ctx, statement, in.Kind, s.principal)
	if err != nil {
		// Scrubbed: the error names the operation, never the content.
		s.log.Error("remember failed", slog.String("error", err.Error()))
		return nil, RememberOutput{}, fmt.Errorf("remember failed")
	}

	// Identifiers only — never the statement text (L8).
	s.log.Info("remembered",
		slog.String(obs.AttrPrincipalID, string(s.principal)),
		slog.String(obs.AttrClaimID, id))

	out := RememberOutput{ID: id, Statement: statement, Stored: true}
	text := fmt.Sprintf("Stored claim %s. See it with `cred log`; reverse it with `cred forget %s`.", id, id)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, out, nil
}

func (s *Server) handleRecall(ctx context.Context, _ *mcp.CallToolRequest, in RecallInput) (
	*mcp.CallToolResult, RecallOutput, error,
) {
	query := strings.TrimSpace(in.Query)
	if query == "" {
		return nil, RecallOutput{}, fmt.Errorf("query must not be empty")
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 5
	}

	res, err := s.recall.Recall(ctx, recall.Request{
		Query:     query,
		Principal: s.principal,
		Limit:     limit,
		Now:       time.Now().UTC(),
	})
	if err != nil {
		// Scrubbed: the error names the operation, never the content. An
		// error that echoes restricted text is an access-control bypass
		// wearing a diagnostic's clothes.
		s.log.Error("recall failed", slog.String("error", err.Error()))
		return nil, RecallOutput{}, fmt.Errorf("recall failed")
	}

	out := RecallOutput{
		AsOf:             res.AsOf.Format(time.RFC3339),
		StalenessSeconds: res.StalenessSeconds,
		Returned:         len(res.Claims),
		Omitted:          res.OmittedForBudget,
		TokensUsed:       res.TokensUsed,
		TokenBudget:      res.TokenBudget,
	}
	for _, sc := range res.Claims {
		rc := RecallClaim{
			ID:         sc.Claim.ID,
			Kind:       string(sc.Claim.Kind),
			Statement:  sc.Claim.Statement,
			Score:      sc.Score,
			Confidence: sc.Claim.Confidence,
		}
		for _, e := range sc.Claim.Evidence {
			rc.Evidence = append(rc.Evidence, RecallEvidence{
				Path:      e.Path,
				LineStart: e.LineStart,
				LineEnd:   e.LineEnd,
				Text:      e.ExtractedText,
			})
		}
		out.Claims = append(out.Claims, rc)
	}

	// Identifiers, counts and durations. Never the query and never the claim
	// text: ingested content is untrusted, and logs are read by tools that
	// were not designed to treat it as such.
	s.log.Info("recall",
		slog.String(obs.AttrPrincipalID, string(s.principal)),
		slog.Int(obs.AttrRecallCandidates, res.Candidates),
		slog.Int(obs.AttrRecallAuthorized, res.Authorized),
		slog.Int(obs.AttrRecallOmitted, res.OmittedForBudget),
		slog.Int64(obs.AttrRecallDurationMS, res.Timings.Total.Milliseconds()))

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: Fence(out)}},
	}, out, nil
}
