package pg

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/canhta/cred/internal/claim"
)

// Hit is one candidate from one retrieval arm: an identifier and its rank.
// Deliberately not a score — RRF consumes ranks, and mixing an unbounded BM25
// score with a bounded cosine one is the fusion bug that reads as noise.
type Hit struct {
	ClaimID string
	Rank    int // 1-based
	Raw     float64
}

// LiveChunk is the change-detection record for one seeded chunk.
type LiveChunk struct {
	EvidenceID string
	ClaimID    string
	SHA256     string
}

// LiveChunks returns the currently-live seeded chunks for a repository, keyed
// by "path\x00ordinal". Re-seeding compares hashes against this and skips what
// has not changed, which is what makes seeding idempotent.
func (s *Store) LiveChunks(ctx context.Context, repo string) (map[string]LiveChunk, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT e.source_path, e.chunk_ordinal, e.id, ce.claim_id, e.content_sha256
		  FROM evidence e
		  JOIN claim_evidence ce ON ce.evidence_id = e.id
		  JOIN claims c ON c.id = ce.claim_id
		 WHERE e.source_repo = $1
		   AND e.superseded_at IS NULL
		   AND c.superseded_at IS NULL`, repo)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()

	out := make(map[string]LiveChunk)
	for rows.Next() {
		var path string
		var ordinal int
		var lc LiveChunk
		var sum []byte
		if err := rows.Scan(&path, &ordinal, &lc.EvidenceID, &lc.ClaimID, &sum); err != nil {
			return nil, translate(err)
		}
		lc.SHA256 = hex.EncodeToString(sum)
		out[chunkKey(path, ordinal)] = lc
	}
	return out, translate(rows.Err())
}

func chunkKey(path string, ordinal int) string {
	return fmt.Sprintf("%s\x00%d", path, ordinal)
}

// ChunkKey is the identity of a seeded chunk within a repository.
func ChunkKey(path string, ordinal int) string { return chunkKey(path, ordinal) }

// SeedRecord is one chunk to write: its evidence, the claim resting on it, its
// embedding, and the grants both carry.
type SeedRecord struct {
	Evidence   claim.Evidence
	Claim      claim.Claim
	Embedding  []float32
	Ordinal    int
	Principals []claim.PrincipalID
}

// InsertSeed writes one chunk in a single transaction: evidence, claim, the
// L1 link, both ACLs, and the embedding.
//
// One transaction because a claim that commits without its evidence is an
// orphan claim, and an embedding that commits without its claim is a row reads
// filter out forever. Both are silent.
func (s *Store) InsertSeed(ctx context.Context, modelID int, r SeedRecord) (claimID string, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", translate(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	sum, err := hex.DecodeString(r.Evidence.ContentSHA256)
	if err != nil {
		return "", fmt.Errorf("decode content hash: %w", err)
	}

	var evidenceID string
	err = tx.QueryRow(ctx, `
		INSERT INTO evidence (source_kind, source_repo, source_path, chunk_ordinal,
		                      line_start, line_end, extracted_text, content_sha256,
		                      valid_from, recorded_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)
		RETURNING id`,
		string(r.Evidence.Kind), r.Evidence.Repo, r.Evidence.Path, r.Ordinal,
		r.Evidence.LineStart, r.Evidence.LineEnd, r.Evidence.ExtractedText, sum,
		r.Evidence.Recorded.From,
	).Scan(&evidenceID)
	if err != nil {
		return "", translate(err)
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO claims (kind, statement, scope_kind, scope_value,
		                    valid_from, recorded_at, confidence,
		                    source_repo, extracted_by_model)
		VALUES ($1,$2,$3,$4,$5,$5,$6,$7,$8)
		RETURNING id`,
		string(r.Claim.Kind), r.Claim.Statement,
		string(r.Claim.Scope.Kind), r.Claim.Scope.Value,
		r.Claim.Recorded.From, r.Claim.Confidence,
		r.Claim.SourceRepo, r.Claim.ExtractedByModel,
	).Scan(&claimID)
	if err != nil {
		return "", translate(err)
	}

	if _, err = tx.Exec(ctx,
		`INSERT INTO claim_evidence (claim_id, evidence_id) VALUES ($1,$2)`,
		claimID, evidenceID); err != nil {
		return "", translate(err)
	}

	for _, p := range r.Principals {
		if _, err = tx.Exec(ctx,
			`INSERT INTO claim_acl (claim_id, principal_id) VALUES ($1,$2)`,
			claimID, string(p)); err != nil {
			return "", translate(err)
		}
		if _, err = tx.Exec(ctx,
			`INSERT INTO evidence_acl (evidence_id, principal_id) VALUES ($1,$2)`,
			evidenceID, string(p)); err != nil {
			return "", translate(err)
		}
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO claim_embeddings (embedding_model_id, claim_id, embedding)
		VALUES ($1, $2, $3::halfvec)`,
		modelID, claimID, encodeHalfvec(r.Embedding)); err != nil {
		return "", translate(err)
	}

	return claimID, translate(tx.Commit(ctx))
}

// SupersedeChunk closes a chunk's evidence and claim in transaction time at
// now, and links the claim to its successor.
//
// Nothing is deleted; things expire. The successor is written first so that a
// crash between the two leaves a duplicate rather than a gap — a gap is a
// query that silently returns nothing.
func (s *Store) SupersedeChunk(ctx context.Context, old LiveChunk, successorClaimID string, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return translate(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		UPDATE evidence SET superseded_at = $2, valid_until = $2
		 WHERE id = $1 AND superseded_at IS NULL`, old.EvidenceID, now); err != nil {
		return translate(err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE claims SET superseded_at = $2, valid_until = $2, superseded_by = $3
		 WHERE id = $1 AND superseded_at IS NULL`,
		old.ClaimID, now, successorClaimID); err != nil {
		return translate(err)
	}
	return translate(tx.Commit(ctx))
}

// DenseSearch returns the nearest live claims by inner-product distance.
//
// The model filter is not optional. A per-row model column that reads ignore
// is worthless, and it is the exact bug shipped by four surveyed projects.
func (s *Store) DenseSearch(ctx context.Context, modelID int, vec []float32, k int) ([]Hit, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, e.embedding::halfvec(384) <#> $2::halfvec(384) AS distance
		  FROM claim_embeddings e
		  JOIN claims c ON c.id = e.claim_id
		 WHERE e.embedding_model_id = $1
		   AND c.superseded_at IS NULL
		 ORDER BY distance
		 LIMIT $3`, modelID, encodeHalfvec(vec), k)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	return collectHits(rows)
}

// LexicalSearch returns live claims whose evidence matches query, ranked by
// ts_rank.
//
// websearch_to_tsquery is used rather than plainto_tsquery because it accepts
// quoted phrases and negation without erroring on operator characters that
// occur constantly in code queries.
func (s *Store) LexicalSearch(ctx context.Context, query string, k int) ([]Hit, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, MAX(ts_rank(e.search_tsv, q)) AS rank
		  FROM evidence e
		  JOIN claim_evidence ce ON ce.evidence_id = e.id
		  JOIN claims c ON c.id = ce.claim_id,
		       websearch_to_tsquery('english', $1) AS q
		 WHERE e.search_tsv @@ q
		   AND e.superseded_at IS NULL
		   AND c.superseded_at IS NULL
		 GROUP BY c.id
		 ORDER BY rank DESC
		 LIMIT $2`, query, k)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	return collectHits(rows)
}

func collectHits(rows pgx.Rows) ([]Hit, error) {
	var out []Hit
	rank := 1
	for rows.Next() {
		var h Hit
		if err := rows.Scan(&h.ClaimID, &h.Raw); err != nil {
			return nil, translate(err)
		}
		h.Rank = rank // 1-based; 0-based shifts every score by ~1.6%
		rank++
		out = append(out, h)
	}
	return out, translate(rows.Err())
}

// LoadClaims returns claims by identifier, without their evidence and without
// any access-control decision. Order is not preserved; the caller ranked them.
func (s *Store) LoadClaims(ctx context.Context, ids []string) ([]claim.Claim, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, kind, statement, scope_kind, scope_value,
		       valid_from, valid_until, recorded_at, superseded_at,
		       confidence, source_repo, extracted_by_model
		  FROM claims WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()

	var out []claim.Claim
	for rows.Next() {
		var c claim.Claim
		var validUntil, supersededAt *time.Time
		if err := rows.Scan(&c.ID, &c.Kind, &c.Statement,
			&c.Scope.Kind, &c.Scope.Value,
			&c.Valid.From, &validUntil, &c.Recorded.From, &supersededAt,
			&c.Confidence, &c.SourceRepo, &c.ExtractedByModel); err != nil {
			return nil, translate(err)
		}
		if validUntil != nil {
			c.Valid.Until = *validUntil
		}
		if supersededAt != nil {
			c.Recorded.Until = *supersededAt
		}
		out = append(out, c)
	}
	return out, translate(rows.Err())
}

// LoadEvidence returns the evidence for each claim, keyed by claim ID.
func (s *Store) LoadEvidence(ctx context.Context, claimIDs []string) (map[string][]claim.Evidence, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ce.claim_id, e.id, e.source_kind, e.source_repo, e.source_path,
		       e.line_start, e.line_end, e.extracted_text,
		       e.valid_from, e.valid_until, e.recorded_at, e.superseded_at
		  FROM claim_evidence ce
		  JOIN evidence e ON e.id = ce.evidence_id
		 WHERE ce.claim_id = ANY($1)
		 ORDER BY e.source_path, e.line_start`, claimIDs)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()

	out := make(map[string][]claim.Evidence)
	for rows.Next() {
		var claimID string
		var e claim.Evidence
		var validUntil, supersededAt *time.Time
		if err := rows.Scan(&claimID, &e.ID, &e.Kind, &e.Repo, &e.Path,
			&e.LineStart, &e.LineEnd, &e.ExtractedText,
			&e.Valid.From, &validUntil, &e.Recorded.From, &supersededAt); err != nil {
			return nil, translate(err)
		}
		if validUntil != nil {
			e.Valid.Until = *validUntil
		}
		if supersededAt != nil {
			e.Recorded.Until = *supersededAt
		}
		out[claimID] = append(out[claimID], e)
	}
	return out, translate(rows.Err())
}

// LoadClaimACLs returns the grants on each claim, keyed by claim ID.
//
// Grants, not answers. Nothing here filters, and no argument names a
// principal — internal/acl decides.
func (s *Store) LoadClaimACLs(ctx context.Context, claimIDs []string) (map[string]claim.ACL, error) {
	return s.loadACLs(ctx, `
		SELECT claim_id, principal_id, expires_at
		  FROM claim_acl WHERE claim_id = ANY($1)`, claimIDs)
}

// LoadEvidenceACLs returns the grants on each piece of evidence, keyed by
// evidence ID.
func (s *Store) LoadEvidenceACLs(ctx context.Context, evidenceIDs []string) (map[string]claim.ACL, error) {
	return s.loadACLs(ctx, `
		SELECT evidence_id, principal_id, expires_at
		  FROM evidence_acl WHERE evidence_id = ANY($1)`, evidenceIDs)
}

func (s *Store) loadACLs(ctx context.Context, query string, ids []string) (map[string]claim.ACL, error) {
	rows, err := s.pool.Query(ctx, query, ids)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()

	out := make(map[string]claim.ACL)
	for rows.Next() {
		var key, principal string
		var expires *time.Time
		if err := rows.Scan(&key, &principal, &expires); err != nil {
			return nil, translate(err)
		}
		g := claim.Grant{Principal: claim.PrincipalID(principal)}
		if expires != nil {
			g.ExpiresAt = *expires
		}
		out[key] = append(out[key], g)
	}
	return out, translate(rows.Err())
}

// Counts reports how many live claims and evidence rows exist, for `cred
// doctor` and the first-run log.
func (s *Store) Counts(ctx context.Context) (claims, evidence int, err error) {
	err = s.pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM claims WHERE superseded_at IS NULL),
		       (SELECT count(*) FROM evidence WHERE superseded_at IS NULL)`,
	).Scan(&claims, &evidence)
	return claims, evidence, translate(err)
}
