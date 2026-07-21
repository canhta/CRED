package seed

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/store/pg"
)

// Store is the subset of the row store seeding needs.
type Store interface {
	PresentModel(ctx context.Context) (id int, name string, dims int, err error)
	LiveChunks(ctx context.Context, repo string) (map[string]pg.LiveChunk, error)
	InsertSeed(ctx context.Context, modelID int, r pg.SeedRecord) (string, error)
	ReplaceSeed(ctx context.Context, modelID int, old pg.LiveChunk, r pg.SeedRecord, now time.Time) (string, error)
}

// Embedder is the ingest side of internal/embed.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	ModelName() string
}

// batchSize is how many chunks are embedded per forward pass. gomlx's own
// documentation scopes the pure-Go backend to batches of roughly 32, and
// measured batching gains above that are in the low single digits.
const batchSize = 16

// Report is what a seeding run did.
type Report struct {
	Repo       string
	Files      int
	Chunks     int
	Inserted   int
	Superseded int
	Unchanged  int
	Duration   time.Duration
}

// Seeder writes documentation chunks as evidence-backed claims.
type Seeder struct {
	store Store
	embed Embedder
	log   *slog.Logger
}

// New builds a Seeder.
func New(store Store, embedder Embedder, log *slog.Logger) *Seeder {
	return &Seeder{store: store, embed: embedder, log: log}
}

// Run seeds from root. It is idempotent: a chunk whose content hash is
// unchanged is skipped without being re-embedded, and a changed chunk
// supersedes its predecessor rather than duplicating it.
//
// Nothing is deleted. A chunk that disappears from a file keeps its
// superseded row, because the evidence for what the project used to say is
// itself worth retaining.
func (s *Seeder) Run(ctx context.Context, root string, principals []claim.PrincipalID) (Report, error) {
	started := time.Now()

	abs, err := filepath.Abs(root)
	if err != nil {
		return Report{}, fmt.Errorf("resolve %s: %w", root, err)
	}
	repo := filepath.ToSlash(abs)

	modelID, modelName, _, err := s.store.PresentModel(ctx)
	if err != nil {
		return Report{}, fmt.Errorf("resolve embedding model: %w", err)
	}
	if modelName != s.embed.ModelName() {
		return Report{}, fmt.Errorf(
			"embedding model mismatch: database holds %q, this binary loaded %q",
			modelName, s.embed.ModelName())
	}

	files, err := Discover(abs)
	if err != nil {
		return Report{}, err
	}

	live, err := s.store.LiveChunks(ctx, repo)
	if err != nil {
		return Report{}, fmt.Errorf("load existing chunks: %w", err)
	}

	rep := Report{Repo: repo, Files: len(files)}
	var pending []Chunk

	flush := func() error {
		if len(pending) == 0 {
			return nil
		}
		texts := make([]string, len(pending))
		for i, c := range pending {
			texts[i] = c.Text
		}
		vecs, err := s.embed.Embed(ctx, texts)
		if err != nil {
			return fmt.Errorf("embed chunks: %w", err)
		}
		if len(vecs) != len(pending) {
			return fmt.Errorf("embed chunks: got %d vectors for %d chunks",
				len(vecs), len(pending))
		}
		now := time.Now().UTC()
		for i, c := range pending {
			key := pg.ChunkKey(c.Path, c.Ordinal)
			old, existed := live[key]

			rec := pg.SeedRecord{
				Ordinal:    c.Ordinal,
				Principals: principals,
				Evidence: claim.Evidence{
					Kind:             claim.SourceDocument,
					Repo:             repo,
					Path:             c.Path,
					LineStart:        c.LineStart,
					LineEnd:          c.LineEnd,
					ExtractedText:    c.Text,
					ContentSHA256:    c.SHA256,
					AnchorSymbolPath: c.Anchor.SymbolPath,
					AnchorNodeHash:   c.Anchor.NodeHash,
					AnchorWindowHash: c.Anchor.WindowHash,
					Valid:            claim.Interval{From: now},
					Recorded:         claim.Interval{From: now},
				},
				Claim: claim.Claim{
					Kind:      claim.KindReference,
					Statement: c.Statement(),
					Scope: claim.Scope{
						Kind:  claim.ScopePath,
						Value: c.Path,
					},
					Valid:    claim.Interval{From: now},
					Recorded: claim.Interval{From: now},
					// Seeded claims start below an attested one. Confidence is
					// additive and explainable; this is the base term and
					// nothing has been added to it yet.
					Confidence:       0.5,
					SourceRepo:       repo,
					ExtractedByModel: "", // deterministic extraction; no model
				},
				Embedding: vecs[i],
			}

			// A changed chunk is superseded and replaced in one transaction, so
			// the new live row never collides with the old one on the
			// one-live-chunk-per-(repo,path,ordinal) index, and a crash leaves
			// neither a duplicate nor a gap. A new chunk is a plain insert.
			var claimID string
			if existed {
				claimID, err = s.store.ReplaceSeed(ctx, modelID, old, rec, now)
				if err != nil {
					return fmt.Errorf("replace %s: %w", key, err)
				}
				rep.Superseded++
			} else {
				claimID, err = s.store.InsertSeed(ctx, modelID, rec)
				if err != nil {
					return err
				}
			}
			rep.Inserted++

			// Never log chunk text. Ingested content is untrusted (L8) and
			// logs are read by tools that were not designed to treat it as
			// such. Identifiers, counts and durations only.
			s.log.Debug("seeded chunk",
				slog.String("path", c.Path),
				slog.Int("ordinal", c.Ordinal),
				slog.Int("line_start", c.LineStart),
				slog.Int("line_end", c.LineEnd),
				slog.String("claim_id", claimID))
		}
		pending = pending[:0]
		return nil
	}

	for _, path := range files {
		chunks, err := ChunkFile(path, abs)
		if err != nil {
			return rep, err
		}
		for _, c := range chunks {
			rep.Chunks++
			if old, ok := live[pg.ChunkKey(c.Path, c.Ordinal)]; ok && old.SHA256 == c.SHA256 {
				rep.Unchanged++
				continue
			}
			pending = append(pending, c)
			if len(pending) >= batchSize {
				if err := flush(); err != nil {
					return rep, err
				}
			}
		}
	}
	if err := flush(); err != nil {
		return rep, err
	}

	rep.Duration = time.Since(started)
	return rep, nil
}
