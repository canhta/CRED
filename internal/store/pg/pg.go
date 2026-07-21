// Package pg is the only package in CRED that imports a database driver.
// depguard fails the build if any other internal package imports pgx.
//
// The store returns rows, never decisions. There is no GetVisibleClaims(principal)
// here and there never will be: no exported function in this package takes a
// principal, because the intersection is computed in Go by internal/acl. That
// costs a round trip of rows Postgres could have discarded, and it buys the
// only version of access control that can be unit-tested.
//
// Errors crossing out of this package are translated. A *pgconn.PgError must
// never escape, or internal/store becomes an API commitment to pgx and the
// layering is defeated through the error channel — the least visible way to
// defeat it.
package pg

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound reports a row that does not exist. It unwraps to sql.ErrNoRows,
// which is what callers outside this package match on — pgx.ErrNoRows is a
// proxy for the same sentinel, so matching the standard one keeps the driver
// out of the domain layer.
var ErrNotFound = fmt.Errorf("cred: not found: %w", sql.ErrNoRows)

// ErrExtensionMissing reports that pgvector is not installed. It is separate
// because it is the failure a new user hits most often and it has a specific,
// printable fix.
var ErrExtensionMissing = errors.New("cred: the pgvector extension is not installed")

// ErrEmailTaken reports a registration attempt against an email that already
// has an account.
var ErrEmailTaken = errors.New("cred: email already registered")

// ErrBootstrapExists reports a registration attempt that lost the race to
// become the account created via the open bootstrap path: another
// registration already committed one. Postgres's
// user_credentials_one_bootstrap partial unique index is what makes this
// detectable at all -- the count-then-insert check in the handler cannot
// see a concurrent, not-yet-committed insert.
var ErrBootstrapExists = errors.New("cred: a bootstrap account already exists")

// Store holds a connection pool.
type Store struct {
	pool *pgxpool.Pool
}

// Open creates a pool against databaseURL.
func Open(ctx context.Context, databaseURL string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	// Bounded by intent rather than by default: one MCP server and one CLI
	// invocation do not need 100 backends, and an idle-in-transaction backend
	// anywhere in the cluster pins the xmin horizon.
	cfg.MaxConns = 8
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", translate(err))
	}
	return &Store{pool: pool}, nil
}

// Close releases the pool.
func (s *Store) Close() { s.pool.Close() }

// Pool exposes the pool to sibling code in this package's tests and to the
// migration runner. It is deliberately not part of any interface consumed
// elsewhere.
func (s *Store) Pool() *pgxpool.Pool { return s.pool }

// translate converts a driver error into one the domain layer can match
// without importing pgx.
func translate(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch {
		case pgErr.Code == "42704" && strings.Contains(pgErr.Message, "vector"),
			pgErr.Code == "42883" && strings.Contains(pgErr.Message, "halfvec"):
			return ErrExtensionMissing
		case pgErr.Code == "23505" && pgErr.ConstraintName == "user_credentials_email_key":
			return ErrEmailTaken
		case pgErr.Code == "23505" && pgErr.ConstraintName == "user_credentials_one_bootstrap":
			return ErrBootstrapExists
		}
		// Carry the constraint name, which is the part a caller can act on,
		// and drop the driver type.
		if pgErr.ConstraintName != "" {
			return fmt.Errorf("cred: constraint %s violated: %s",
				pgErr.ConstraintName, pgErr.Message)
		}
		return fmt.Errorf("cred: postgres %s: %s", pgErr.Code, pgErr.Message)
	}
	return err
}

// Version reports the server version string.
func (s *Store) Version(ctx context.Context) (string, error) {
	var v string
	err := s.pool.QueryRow(ctx, `SELECT version()`).Scan(&v)
	return v, translate(err)
}

// VectorExtension reports the installed pgvector version.
func (s *Store) VectorExtension(ctx context.Context) (string, error) {
	var v string
	err := s.pool.QueryRow(ctx,
		`SELECT extversion FROM pg_extension WHERE extname = 'vector'`).Scan(&v)
	if errors.Is(translate(err), ErrNotFound) {
		return "", ErrExtensionMissing
	}
	return v, translate(err)
}

// PresentModel returns the embedding model reads must filter on.
func (s *Store) PresentModel(ctx context.Context) (id int, name string, dims int, err error) {
	err = s.pool.QueryRow(ctx,
		`SELECT id, name, dimensions FROM embedding_models WHERE status = 'PRESENT'`,
	).Scan(&id, &name, &dims)
	return id, name, dims, translate(err)
}

// hashArg turns a hex hash into a bytea insert argument, or NULL when empty. An
// empty tier-2/3 hash means "no anchor at this tier" — an attestation, or a
// document span with no enclosing heading — and NULL is how the schema records
// that, distinct from a zero-length hash.
func hashArg(hexStr string) (any, error) {
	if hexStr == "" {
		return nil, nil
	}
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf("decode anchor hash: %w", err)
	}
	return b, nil
}

// encodeHalfvec renders a vector in pgvector's text input format.
//
// Thirty lines rather than a module. pgvector-go exists and works, but the
// only thing needed here is one text encoding, and every dependency is a
// maintenance liability for a solo maintainer.
func encodeHalfvec(v []float32) string {
	var b strings.Builder
	b.Grow(len(v)*10 + 2)
	b.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(f), 'g', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}
