// Package curate holds the deterministic write side of CRED and the River
// workers that run it off the turn.
//
// It is the other half of the LLM boundary. internal/nominate proposes;
// internal/curate decides and writes (L2). Everything a model produced is
// untrusted until it reaches here: the executor resolves each candidate's quote
// against the trusted source, drops any it cannot find (L1), and stores the
// resolved span — never the model's free text — as evidence (L8).
//
// This package imports the store; internal/nominate may not. That asymmetry is
// the structural form of "the model nominates, code decides".
package curate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/nominate"
	"github.com/canhta/cred/internal/store/pg"
)

// WriteStore is the subset of the row store the executor writes through.
type WriteStore interface {
	PresentModel(ctx context.Context) (id int, name string, dims int, err error)
	WriteClaim(ctx context.Context, modelID int, r pg.WriteRecord) (string, error)
}

// Embedder is the write side of internal/embed. Written claims are embedded so
// the dense arm can retrieve them; an unembedded claim is invisible to recall.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	ModelName() string
}

// Executor turns validated candidates and human attestations into rows. This is
// where "code decides" lives.
type Executor struct {
	store WriteStore
	embed Embedder
	log   *slog.Logger
}

// NewExecutor builds an Executor.
func NewExecutor(store WriteStore, embed Embedder, log *slog.Logger) *Executor {
	return &Executor{store: store, embed: embed, log: log}
}

// WriteResult reports what a nomination write did. DroppedNoEvidence is the L1
// count: candidates whose quote did not resolve against the source. It must be
// visible, because a silently dropped candidate and a silently stored one are
// the two failures the whole boundary exists to prevent.
type WriteResult struct {
	Written           []string
	DroppedNoEvidence int
}

// WriteCandidates writes the candidates that resolve against in.Source. A
// candidate whose quote is not a verbatim span of the source is dropped (L1):
// the model pointed at evidence that does not exist, so there is no claim.
//
// The evidence stored is the resolved span from the source, not the model's
// text. Statements are the model's; evidence is the source's. That division is
// the L8 guarantee.
func (e *Executor) WriteCandidates(ctx context.Context, in nominate.Input, cands []nominate.Candidate) (WriteResult, error) {
	modelID, err := e.presentModel(ctx)
	if err != nil {
		return WriteResult{}, err
	}

	var res WriteResult
	for _, c := range cands {
		span, ok := locate(in.Source, c.Quote, in.BaseLine)
		if !ok {
			// L1: no resolvable evidence, no claim. Never log the quote or the
			// statement — ingested content is untrusted (L8).
			res.DroppedNoEvidence++
			e.debug("dropped candidate with unresolvable evidence",
				slog.String("path", in.Path), slog.String("trigger", in.Trigger))
			continue
		}

		now := time.Now().UTC()
		id, err := e.writeOne(ctx, modelID, writeInput{
			statement:  c.Statement,
			kind:       c.Kind,
			scope:      in.Scope,
			sourceKind: in.SourceKind,
			repo:       in.Repo,
			path:       in.Path,
			lineStart:  span.lineStart,
			lineEnd:    span.lineEnd,
			evidence:   span.text,
			confidence: c.Confidence,
			model:      nominate.PromptVersion, // the boundary that produced it
			principals: in.Principals,
			now:        now,
		})
		if err != nil {
			return res, err
		}
		res.Written = append(res.Written, id)
	}
	return res, nil
}

// Attest writes a human attestation — the explicit write path, used by `cred
// remember` and the MCP remember tool. It is deterministic and takes no model:
// L1 counts human attestation as evidence, so the statement is its own evidence
// and the principal is the attester. Because it never calls an LLM, the explicit
// write path needs no API key — the same zero-config property the read path has.
//
// kind may be empty (defaults to Reference); an unrecognized kind is rejected
// rather than coerced, because kind determines validity semantics and guessing
// one is the kind of silent decision this project refuses to make.
func (e *Executor) Attest(ctx context.Context, statement, kind string, principal claim.PrincipalID) (string, error) {
	if strings.TrimSpace(statement) == "" {
		return "", fmt.Errorf("attestation statement must not be empty")
	}
	if principal == "" {
		return "", fmt.Errorf("attestation must name a principal")
	}
	k, err := parseKind(kind)
	if err != nil {
		return "", err
	}
	modelID, err := e.presentModel(ctx)
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	return e.writeOne(ctx, modelID, writeInput{
		statement:  statement,
		kind:       k,
		scope:      claim.Scope{Kind: claim.ScopeOrg, Value: "attested"},
		sourceKind: claim.SourceAttestation,
		repo:       "",
		path:       "attestation",
		lineStart:  1,
		lineEnd:    1,
		evidence:   statement, // the assertion is its own evidence
		confidence: attestedConfidence,
		model:      "attestation",
		attestedBy: principal,
		principals: []claim.PrincipalID{principal},
		now:        now,
	})
}

// parseKind maps a string to a claim kind. Empty means Reference; anything else
// must be one of the closed set.
func parseKind(kind string) (claim.Kind, error) {
	if kind == "" {
		return claim.KindReference, nil
	}
	k := claim.Kind(kind)
	switch k {
	case claim.KindConvention, claim.KindDecision, claim.KindConstraint,
		claim.KindRejectedApproach, claim.KindFailure, claim.KindReference:
		return k, nil
	default:
		return "", fmt.Errorf("unknown claim kind %q: one of Convention, Decision, "+
			"Constraint, RejectedApproach, Failure, Reference", kind)
	}
}

// attestedConfidence is the base score for a human-attested claim. It sits above
// a seeded claim's 0.5: a person asserting something outweighs a document chunk,
// though the score is additive and explainable, never a posterior.
const attestedConfidence = 0.7

type writeInput struct {
	statement  string
	kind       claim.Kind
	scope      claim.Scope
	sourceKind claim.SourceKind
	repo       string
	path       string
	lineStart  int
	lineEnd    int
	evidence   string
	confidence float64
	model      string
	attestedBy claim.PrincipalID
	principals []claim.PrincipalID
	now        time.Time
}

func (e *Executor) writeOne(ctx context.Context, modelID int, w writeInput) (string, error) {
	vecs, err := e.embed.Embed(ctx, []string{w.statement})
	if err != nil {
		return "", fmt.Errorf("embed statement: %w", err)
	}
	if len(vecs) != 1 {
		return "", fmt.Errorf("embed statement: got %d vectors, want 1", len(vecs))
	}

	ev := claim.Evidence{
		Kind:          w.sourceKind,
		Repo:          w.repo,
		Path:          w.path,
		LineStart:     w.lineStart,
		LineEnd:       w.lineEnd,
		ExtractedText: w.evidence,
		ContentSHA256: hashHex(w.evidence),
		Valid:         claim.Interval{From: w.now},
		Recorded:      claim.Interval{From: w.now},
	}
	if w.attestedBy != "" {
		ev.AttestedBy = w.attestedBy
		ev.AttestedAt = w.now
	}

	return e.store.WriteClaim(ctx, modelID, pg.WriteRecord{
		Evidence: ev,
		Claim: claim.Claim{
			Kind:             w.kind,
			Statement:        w.statement,
			Scope:            w.scope,
			Valid:            claim.Interval{From: w.now},
			Recorded:         claim.Interval{From: w.now},
			Confidence:       w.confidence,
			SourceRepo:       w.repo,
			ExtractedByModel: w.model,
			PromptVersion:    nominate.PromptVersion,
		},
		Embedding:       vecs[0],
		StatementSHA256: hashHex(normalizeStatement(w.statement)),
		Principals:      w.principals,
	})
}

func (e *Executor) presentModel(ctx context.Context) (int, error) {
	modelID, modelName, _, err := e.store.PresentModel(ctx)
	if err != nil {
		return 0, fmt.Errorf("resolve embedding model: %w", err)
	}
	if modelName != e.embed.ModelName() {
		return 0, fmt.Errorf(
			"embedding model mismatch: database holds %q, this binary loaded %q",
			modelName, e.embed.ModelName())
	}
	return modelID, nil
}

func (e *Executor) debug(msg string, attrs ...any) {
	if e.log != nil {
		e.log.Debug(msg, attrs...)
	}
}

func hashHex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
