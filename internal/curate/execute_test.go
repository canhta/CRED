package curate

import (
	"context"
	"testing"

	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/nominate"
	"github.com/canhta/cred/internal/store/pg"
)

// fakeStore records what the executor writes, without a database. It lets the
// L1 and L2 write-side rules be tested as pure functions over the executor.
type fakeStore struct {
	written []pg.WriteRecord
}

func (f *fakeStore) PresentModel(context.Context) (int, string, int, error) {
	return 1, "test-model", 384, nil
}

func (f *fakeStore) WriteClaim(_ context.Context, _ int, r pg.WriteRecord) (string, error) {
	f.written = append(f.written, r)
	return "claim-" + r.Claim.Statement, nil
}

type fakeEmbed struct{}

func (fakeEmbed) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range out {
		out[i] = make([]float32, 384)
	}
	return out, nil
}
func (fakeEmbed) ModelName() string { return "test-model" }

func TestWriteCandidatesDropsEvidencelessCandidate(t *testing.T) {
	// L1: a candidate whose quote is not a span of the trusted source has no
	// evidence, so there is no claim. The one that resolves is written; the one
	// that does not is dropped, and the drop is counted, never silent.
	store := &fakeStore{}
	exec := NewExecutor(store, fakeEmbed{}, nil)

	in := nominate.Input{
		Source:     "we chose reciprocal rank fusion at k equal to sixty\n",
		SourceKind: claim.SourceDocument,
		Repo:       "r", Path: "AGENTS.md", BaseLine: 1,
		Scope:      claim.Scope{Kind: claim.ScopePath, Value: "AGENTS.md"},
		Principals: []claim.PrincipalID{"local"},
	}
	cands := []nominate.Candidate{
		{Kind: claim.KindDecision, Statement: "RRF at k=60",
			Quote: "reciprocal rank fusion at k equal to sixty", Confidence: 0.9},
		{Kind: claim.KindDecision, Statement: "invented claim",
			Quote: "this text is nowhere in the source", Confidence: 0.9},
	}

	res, err := exec.WriteCandidates(context.Background(), in, cands)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Written) != 1 {
		t.Fatalf("wrote %d claims, want 1", len(res.Written))
	}
	if res.DroppedNoEvidence != 1 {
		t.Fatalf("dropped %d, want 1 (L1)", res.DroppedNoEvidence)
	}
	if len(store.written) != 1 {
		t.Fatalf("store received %d writes, want 1", len(store.written))
	}

	rec := store.written[0]
	// The evidence stored is the resolved span from the SOURCE, not the model's
	// free text — the L8 guarantee. Statement is the model's; evidence is the
	// source's.
	if rec.Evidence.ExtractedText != "reciprocal rank fusion at k equal to sixty" {
		t.Fatalf("evidence text = %q, want the source span", rec.Evidence.ExtractedText)
	}
	if rec.Claim.Statement != "RRF at k=60" {
		t.Fatalf("claim statement = %q, want the candidate statement", rec.Claim.Statement)
	}
	if rec.Claim.ExtractedByModel == "" {
		t.Fatal("a written claim must record what produced it (provenance, L8)")
	}
	if rec.StatementSHA256 == "" {
		t.Fatal("a written claim must carry its dedup hash")
	}
}

func TestAttestWritesHumanEvidence(t *testing.T) {
	// L1 counts human attestation as evidence: the statement is its own
	// evidence, the principal is the attester, and no model is called — so the
	// explicit path needs no key.
	store := &fakeStore{}
	exec := NewExecutor(store, fakeEmbed{}, nil)

	id, err := exec.Attest(context.Background(), "prefer pgx over an ORM", "Decision", "alice")
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("attest returned no id")
	}
	if len(store.written) != 1 {
		t.Fatalf("store received %d writes, want 1", len(store.written))
	}
	rec := store.written[0]
	if rec.Evidence.Kind != claim.SourceAttestation {
		t.Fatalf("evidence kind = %q, want attestation", rec.Evidence.Kind)
	}
	if rec.Evidence.AttestedBy != "alice" {
		t.Fatalf("attested_by = %q, want alice", rec.Evidence.AttestedBy)
	}
	if rec.Evidence.ExtractedText != "prefer pgx over an ORM" {
		t.Fatal("the assertion must be its own evidence")
	}
	if rec.Claim.Kind != claim.KindDecision {
		t.Fatalf("claim kind = %q, want Decision", rec.Claim.Kind)
	}
}

func TestAttestRejectsUnknownKind(t *testing.T) {
	exec := NewExecutor(&fakeStore{}, fakeEmbed{}, nil)
	_, err := exec.Attest(context.Background(), "x", "Opinion", "alice")
	if err == nil {
		t.Fatal("an unknown kind must be rejected, not coerced")
	}
}

func TestAttestRejectsEmptyStatement(t *testing.T) {
	exec := NewExecutor(&fakeStore{}, fakeEmbed{}, nil)
	if _, err := exec.Attest(context.Background(), "   ", "", "alice"); err == nil {
		t.Fatal("an empty statement must be rejected")
	}
}
