package main

import (
	"context"
	"log/slog"
	"os"

	"olexsmir.xyz/mugit/internal/cli"
)

// NOTE: sets during build
// go build -ldflags="-X 'main.version=v1.0.0'"
var version = "develop"

func main() {
	if err := cli.New(version).Run(context.Background(), os.Args); err != nil {
		slog.Error("mugit", "err", err)
		os.Exit(1)
	}
}
