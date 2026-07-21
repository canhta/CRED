package pg

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/canhta/cred/internal/claim"
)

// WriteRecord is one claim the write path is about to store: its evidence, the
// claim resting on it, its embedding, the normalized-statement hash used for
// exact-hash deduplication, and the grants both carry.
//
// This is the write-path sibling of SeedRecord. It differs in what it records:
// provenance (extracted_by_model, prompt_version, source_commit) and the
// statement hash, none of which a deterministic seed produces.
type WriteRecord struct {
	Evidence        claim.Evidence
	Claim           claim.Claim
	Embedding       []float32
	StatementSHA256 string // hex of the normalized statement hash
	Principals      []claim.PrincipalID
}

// WriteClaim writes one claim in a single transaction: evidence, claim, the L1
// link, both ACLs, and the embedding.
//
// One transaction, for the same reason InsertSeed is: a claim that commits
// without its evidence is an orphan (L1 violated through a crash rather than a
// bug), and an embedding without its claim is a row reads filter out forever.
//
// This is where "code decides" lands (L2). The nominator proposed a candidate;
// by the time control reaches here the candidate has been validated and its
// evidence resolved against the trusted source. Nothing the model returned is
// trusted past this point — the evidence text stored is the span from the
// source, not free text from the model.
func (s *Store) WriteClaim(ctx context.Context, modelID int, r WriteRecord) (claimID string, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", translate(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	evidenceSum, err := hex.DecodeString(r.Evidence.ContentSHA256)
	if err != nil {
		return "", fmt.Errorf("decode evidence hash: %w", err)
	}
	statementSum, err := hex.DecodeString(r.StatementSHA256)
	if err != nil {
		return "", fmt.Errorf("decode statement hash: %w", err)
	}

	// Attestation carries a person, not a repository span. The schema requires
	// attested_by and attested_at to be set together, and rejects either alone.
	var attestedBy any
	var attestedAt any
	if r.Evidence.AttestedBy != "" {
		attestedBy = string(r.Evidence.AttestedBy)
		attestedAt = r.Evidence.AttestedAt
	}

	var evidenceID string
	err = tx.QueryRow(ctx, `
		INSERT INTO evidence (source_kind, source_repo, source_path, chunk_ordinal,
		                      line_start, line_end, extracted_text, content_sha256,
		                      attested_by, attested_at, valid_from, recorded_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11)
		RETURNING id`,
		// chunk_ordinal is NULL: write-path evidence is not a seeded chunk, so it
		// is exempt from the one-live-chunk-per-(repo,path,ordinal) uniqueness.
		// Many written claims may point at spans of the same file.
		string(r.Evidence.Kind), r.Evidence.Repo, r.Evidence.Path, nil,
		r.Evidence.LineStart, r.Evidence.LineEnd, r.Evidence.ExtractedText, evidenceSum,
		attestedBy, attestedAt, r.Evidence.Recorded.From,
	).Scan(&evidenceID)
	if err != nil {
		return "", translate(err)
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO claims (kind, statement, scope_kind, scope_value,
		                    valid_from, recorded_at, confidence, statement_sha256,
		                    source_repo, source_commit, extracted_by_model, prompt_version)
		VALUES ($1,$2,$3,$4,$5,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id`,
		string(r.Claim.Kind), r.Claim.Statement,
		string(r.Claim.Scope.Kind), r.Claim.Scope.Value,
		r.Claim.Recorded.From, r.Claim.Confidence, statementSum,
		// source_commit stays empty: a written claim's evidence is a live span
		// or an attestation, both of which can change — that mutability is the
		// point (D-016), and an immutable commit hash would defeat it.
		r.Claim.SourceRepo, "", r.Claim.ExtractedByModel, r.Claim.PromptVersion,
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

// DupMember is one claim in a group that shares a normalized statement.
type DupMember struct {
	ID         string
	RecordedAt time.Time
}

// LiveDuplicateGroups returns the live claims whose normalized statement hash
// is shared by more than one claim, grouped by that hash.
//
// The store returns the groups; internal/curate decides which member survives
// and internal/temporal closes the losers' intervals. The decision is not made
// in SQL, for the same reason L5 is not: a deterministic reconciler that runs
// in Go can be unit-tested and produces byte-identical output across runs.
func (s *Store) LiveDuplicateGroups(ctx context.Context) ([][]DupMember, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT statement_sha256, id, recorded_at
		  FROM claims
		 WHERE superseded_at IS NULL
		   AND statement_sha256 IS NOT NULL
		   AND statement_sha256 IN (
		       SELECT statement_sha256 FROM claims
		        WHERE superseded_at IS NULL AND statement_sha256 IS NOT NULL
		        GROUP BY statement_sha256 HAVING count(*) > 1)
		 ORDER BY statement_sha256, recorded_at, id`)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()

	groups := make(map[string][]DupMember)
	var order []string
	for rows.Next() {
		var hash []byte
		var m DupMember
		if err := rows.Scan(&hash, &m.ID, &m.RecordedAt); err != nil {
			return nil, translate(err)
		}
		key := hex.EncodeToString(hash)
		if _, seen := groups[key]; !seen {
			order = append(order, key)
		}
		groups[key] = append(groups[key], m)
	}
	if err := translate(rows.Err()); err != nil {
		return nil, err
	}

	out := make([][]DupMember, 0, len(order))
	for _, k := range order {
		out = append(out, groups[k])
	}
	return out, nil
}

// ApplySupersession persists a supersession that internal/temporal already
// computed on the incumbent claim: it writes the closed transaction interval,
// the closed validity interval, the successor edge, and the reason.
//
// The caller passes the claim as internal/temporal.Supersede returned it, so
// the half-open closing lives in one tested place and is not reimplemented in
// SQL. The WHERE guard keeps the update idempotent under a retried River job:
// a second attempt on an already-superseded row changes nothing.
func (s *Store) ApplySupersession(ctx context.Context, c claim.Claim, reason string) error {
	var validUntil any
	if !c.Valid.Until.IsZero() {
		validUntil = c.Valid.Until
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE claims
		   SET superseded_at = $2, valid_until = $3, superseded_by = $4,
		       supersede_reason = $5
		 WHERE id = $1 AND superseded_at IS NULL`,
		c.ID, c.Recorded.Until, validUntil, c.SupersededBy, reason)
	if err != nil {
		return translate(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ForgetClaim expires a claim at now with no successor (D-016 reversal). It is
// expiry, not supersession: nothing replaces the claim, so superseded_by stays
// NULL and the reason is 'forgotten'. Nothing is deleted — the row remains,
// closed in transaction time, so the record that it was once believed survives.
func (s *Store) ForgetClaim(ctx context.Context, id string, now time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE claims
		   SET superseded_at = $2, valid_until = $2, supersede_reason = 'forgotten'
		 WHERE id = $1 AND superseded_at IS NULL`, id, now)
	if err != nil {
		return translate(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// WriteEntry is one row in `cred log`: a written claim with enough provenance
// to see what it is, where it came from, and whether it is still live.
type WriteEntry struct {
	ID              string
	Kind            string
	Statement       string
	ScopeKind       string
	ScopeValue      string
	ExtractedBy     string
	RecordedAt      time.Time
	SupersededAt    *time.Time
	SupersedeReason string
}

// RecentWrites returns the most recently recorded claims, newest first,
// including superseded ones so the log shows the whole write history — a write
// that was later forgotten or deduplicated is exactly what an operator needs to
// see. Seeded claims (extracted_by_model = ”) are excluded: the log is the
// write path's surface, not the seed's.
func (s *Store) RecentWrites(ctx context.Context, limit int) ([]WriteEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, kind, statement, scope_kind, scope_value,
		       extracted_by_model, recorded_at, superseded_at,
		       coalesce(supersede_reason, '')
		  FROM claims
		 WHERE extracted_by_model <> ''
		 ORDER BY recorded_at DESC, id
		 LIMIT $1`, limit)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()

	var out []WriteEntry
	for rows.Next() {
		var e WriteEntry
		if err := rows.Scan(&e.ID, &e.Kind, &e.Statement, &e.ScopeKind, &e.ScopeValue,
			&e.ExtractedBy, &e.RecordedAt, &e.SupersededAt, &e.SupersedeReason); err != nil {
			return nil, translate(err)
		}
		out = append(out, e)
	}
	return out, translate(rows.Err())
}

// LoadClaimForSupersession loads one claim by id, without evidence or ACLs, for
// the reconciler to hand to internal/temporal. It returns ErrNotFound if the
// claim does not exist or is already superseded — a superseded claim cannot be
// superseded again, and the reconciler must see that as a no-op, not a target.
func (s *Store) LoadClaimForSupersession(ctx context.Context, id string) (claim.Claim, error) {
	var c claim.Claim
	var validUntil *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT id, kind, statement, scope_kind, scope_value,
		       valid_from, valid_until, recorded_at
		  FROM claims WHERE id = $1 AND superseded_at IS NULL`, id).Scan(
		&c.ID, &c.Kind, &c.Statement, &c.Scope.Kind, &c.Scope.Value,
		&c.Valid.From, &validUntil, &c.Recorded.From)
	if err != nil {
		return claim.Claim{}, translate(err)
	}
	if validUntil != nil {
		c.Valid.Until = *validUntil
	}
	return c, nil
}
