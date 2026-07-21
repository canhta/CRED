// Command cred is evidence-governed memory for AI agents.
//
// A claim lives only while its evidence does.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/canhta/cred/internal/cli"
)

func main() {
	// The signal context is cancelled on the first signal; a second one is
	// left to the runtime, so a hung shutdown is still interruptible.
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()

	// nil without the embed tag; the web command then serves web/dist from disk
	// or a stub. With -tags embed it is the built SPA baked into the binary.
	cli.WebAssets = webAssets()

	os.Exit(cli.Run(ctx, os.Args, os.Stdin, os.Stdout, os.Stderr))
}
