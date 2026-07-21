package pg

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

// River is the first scheduled work in CRED (D-017). It lives here, in the one
// package permitted to import the driver, because the River client is typed on
// pgx.Tx: constructing it anywhere else would drag the driver across the
// boundary depguard exists to hold. internal/curate names none of these types —
// it takes the client back through narrow interfaces (Insert, Start, Stop).
//
// River's tables are created by its own migrator rather than a goose migration,
// so River owns and versions its schema. `cred migrate` runs both.

// MigrateRiver applies River's schema migrations.
func (s *Store) MigrateRiver(ctx context.Context) error {
	migrator, err := rivermigrate.New(riverpgxv5.New(s.pool), nil)
	if err != nil {
		return fmt.Errorf("river migrator: %w", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return fmt.Errorf("river migrate: %w", translate(err))
	}
	return nil
}

// RiverInsertClient returns an insert-only River client — no queues, no
// workers, never started. It is what `cred capture` uses to enqueue an
// automatic-nomination job and return immediately, so the agent's turn is never
// blocked on extraction (D-017).
func (s *Store) RiverInsertClient() (*river.Client[pgx.Tx], error) {
	c, err := river.NewClient(riverpgxv5.New(s.pool), &river.Config{})
	if err != nil {
		return nil, fmt.Errorf("river insert client: %w", err)
	}
	return c, nil
}

// RiverWorkerClient returns a client configured to work the given registry.
// `cred curate` starts it; it is where the LLM boundary is finally crossed, off
// the turn.
//
// MaxAttempts defaults to 25 in River, whose exponential schedule stretches to
// roughly three weeks — meaningless for a nomination job. The worker-ops spike
// sets it to 3–5; 5 here.
func (s *Store) RiverWorkerClient(workers *river.Workers, log *slog.Logger) (*river.Client[pgx.Tx], error) {
	c, err := river.NewClient(riverpgxv5.New(s.pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 4},
		},
		Workers:     workers,
		Logger:      log,
		MaxAttempts: 5,
	})
	if err != nil {
		return nil, fmt.Errorf("river worker client: %w", err)
	}
	return c, nil
}
