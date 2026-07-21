package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/canhta/cred/internal/config"
)

func runMigrate(ctx context.Context, args []string, cfg config.Config, out io.Writer) error {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w: %w", ErrUsage, err)
	}

	st, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer st.Close()

	res, err := st.Migrate(ctx)
	if err != nil {
		// A partial migration is reported with the version the database
		// actually reached. Migration failure is not atomic, and an operator
		// told only "it failed" has to go and look.
		return fmt.Errorf("migrate: %w", err)
	}

	if res.FromVersion == res.ToVersion {
		fmt.Fprintf(out, "migrate    already at version %d, nothing to apply\n", res.ToVersion)
		return nil
	}
	fmt.Fprintf(out, "migrate    version %d -> %d\n", res.FromVersion, res.ToVersion)
	return nil
}
