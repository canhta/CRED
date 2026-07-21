package curate

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/nominate"
)

// NominateArgs is the automatic-write job: material a trigger captured, to be
// extracted off the turn. It carries the source verbatim so the worker re-embeds
// and re-extracts; nothing is precomputed on the turn, which is the whole point
// of pushing it to a worker (D-017).
//
// It carries no claim and no evidence — only the trusted source and its
// context. The worker nominates from it and code decides what, if anything, is
// written.
type NominateArgs struct {
	Source     string   `json:"source"`
	SourceKind string   `json:"source_kind"`
	Repo       string   `json:"repo"`
	Path       string   `json:"path"`
	BaseLine   int      `json:"base_line"`
	ScopeKind  string   `json:"scope_kind"`
	ScopeValue string   `json:"scope_value"`
	Principals []string `json:"principals"`
	Trigger    string   `json:"trigger"`
}

// Kind names the job for River.
func (NominateArgs) Kind() string { return "cred.nominate" }

// DedupArgs is the exact-hash deduplication pass. It carries nothing: dedup
// scans the whole live set. It is enqueued after a nomination write, uniquely,
// so a burst of writes coalesces into a single pending pass rather than one per
// write.
type DedupArgs struct{}

// Kind names the job for River.
func (DedupArgs) Kind() string { return "cred.dedup" }

// PruneArgs is the scope-growth bound's job: check one scope's live-claim count
// and prune it back if it is over its ceiling (PRD 8). It carries the scope a
// write just landed in, because that is the only scope whose count changed.
type PruneArgs struct {
	ScopeKind  string `json:"scope_kind"`
	ScopeValue string `json:"scope_value"`
}

// Kind names the job for River.
func (PruneArgs) Kind() string { return "cred.prune" }

func (a NominateArgs) toInput() nominate.Input {
	ps := make([]claim.PrincipalID, len(a.Principals))
	for i, p := range a.Principals {
		ps[i] = claim.PrincipalID(p)
	}
	sk := claim.SourceKind(a.SourceKind)
	if sk == "" {
		sk = claim.SourceDocument
	}
	return nominate.Input{
		Source:     a.Source,
		SourceKind: sk,
		Repo:       a.Repo,
		Path:       a.Path,
		BaseLine:   a.BaseLine,
		Scope:      claim.Scope{Kind: claim.ScopeKind(a.ScopeKind), Value: a.ScopeValue},
		Principals: ps,
		Trigger:    a.Trigger,
	}
}

// NominateWorker extracts and writes one captured source, then schedules a dedup
// pass. It is the first thing in CRED to cross the LLM boundary, and it does so
// only here, in the background, never on the agent's turn.
type NominateWorker struct {
	river.WorkerDefaults[NominateArgs]
	nom    nominate.Nominator
	exec   *Executor
	limits *Limiter // nil disables the write-path quota/cost gate
	queue  Queue
	log    *slog.Logger
}

// Work runs the gate → nominate → validate → write → schedule-curation pipeline.
func (w *NominateWorker) Work(ctx context.Context, job *river.Job[NominateArgs]) error {
	in := job.Args.toInput()

	// Contribution quota and cost ceiling, before an LLM call is made (PRD 8).
	// A denial here is loud and recorded (see Limiter.deny), never a silent drop
	// — that loudness is what stops the off-the-turn write path from hiding a
	// poisoning attempt (D-017, L8). It returns nil, not an error: the write was
	// correctly refused, and a retry would only re-deny.
	if w.limits != nil {
		if principal := firstPrincipal(in.Principals); principal != "" {
			ok, err := w.limits.Admit(ctx, principal, in.Scope)
			if err != nil {
				return err // transient counting failure: let River retry
			}
			if !ok {
				return nil // denied, loudly and on the record
			}
		}
	}

	cands, err := w.nom.Nominate(ctx, in)
	if err != nil {
		return err // River retries; a transient model failure is not a lost write
	}

	res, err := w.exec.WriteCandidates(ctx, in, cands)
	if err != nil {
		return err
	}

	// Identifiers and counts only — never the source or the statements (L8).
	if w.log != nil {
		w.log.Info("nominated",
			slog.String("trigger", in.Trigger),
			slog.Int("written", len(res.Written)),
			slog.Int("dropped_no_evidence", res.DroppedNoEvidence))
	}

	if len(res.Written) > 0 && w.queue != nil {
		// Best-effort: a curation pass that fails to enqueue is not a failed
		// write. The next write reschedules dedup, which is idempotent, and the
		// scope-growth bound is re-checked on the next write into the scope.
		_ = Enqueue(ctx, w.queue, DedupArgs{})
		_ = Enqueue(ctx, w.queue, PruneArgs{
			ScopeKind: string(in.Scope.Kind), ScopeValue: in.Scope.Value,
		})
	}
	return nil
}

// DedupWorker runs the reconciler.
type DedupWorker struct {
	river.WorkerDefaults[DedupArgs]
	rec *Reconciler
}

// Work runs one exact-hash deduplication pass.
func (w *DedupWorker) Work(ctx context.Context, _ *river.Job[DedupArgs]) error {
	_, err := w.rec.Dedup(ctx)
	return err
}

// PruneWorker runs the scope-growth bound over one scope.
type PruneWorker struct {
	river.WorkerDefaults[PruneArgs]
	pruner *Pruner
}

// Work prunes one scope back under its ceiling if it is over.
func (w *PruneWorker) Work(ctx context.Context, job *river.Job[PruneArgs]) error {
	if w.pruner == nil {
		return nil
	}
	_, err := w.pruner.Prune(ctx, claim.Scope{
		Kind: claim.ScopeKind(job.Args.ScopeKind), Value: job.Args.ScopeValue,
	})
	return err
}

// Register builds the River worker registry for the curate process. limiter and
// pruner may be nil — the write path then runs without the section 8 controls,
// which is what the read-only and no-key deployments want.
func Register(nom nominate.Nominator, exec *Executor, rec *Reconciler,
	pruner *Pruner, limiter *Limiter, queue Queue, log *slog.Logger,
) *river.Workers {
	workers := river.NewWorkers()
	river.AddWorker(workers, &NominateWorker{nom: nom, exec: exec, limits: limiter, queue: queue, log: log})
	river.AddWorker(workers, &DedupWorker{rec: rec})
	river.AddWorker(workers, &PruneWorker{pruner: pruner})
	return workers
}

// Queue is the narrow insert surface curate needs. *river.Client[pgx.Tx]
// satisfies it structurally, which is how the driver stays inside
// internal/store/pg: this package never names a pgx type.
type Queue interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// Runner is the narrow lifecycle surface curate needs to drive the worker
// client. *river.Client[pgx.Tx] satisfies it.
type Runner interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// EnqueueNominate schedules an automatic-write job. This is the entry point an
// agent hook calls (through `cred capture`); it returns as soon as the job is
// durably enqueued, so the agent's turn is never blocked on extraction.
func EnqueueNominate(ctx context.Context, q Queue, args NominateArgs) error {
	return Enqueue(ctx, q, args)
}

// Enqueue inserts a job.
//
// Each nomination write enqueues its own dedup pass, without job-level
// uniqueness. Coalescing bursts with a unique-job window was tried and cut: it
// races — a pending pass from an earlier write can run *before* a later write
// commits and then suppress the pass that would have caught the later
// write's duplicate. A dedup pass is a cheap scan and idempotent, so running one
// per write is correct where coalescing was merely cheaper-and-wrong.
func Enqueue(ctx context.Context, q Queue, args river.JobArgs) error {
	_, err := q.Insert(ctx, args, nil)
	return err
}
