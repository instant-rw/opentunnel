package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/opentunnel/opentunnel/cli/internal/app"
)

var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	application, err := app.New(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "opentunnel: %v\n", err)
		os.Exit(1)
	}
	if err := application.Run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "opentunnel: %v\n", err)
		os.Exit(1)
	}
}
