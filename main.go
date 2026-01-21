package main

import (
	"context"
	"log/slog"
	"os"

	"olexsmir.xyz/mugit/internal/cli"
)

func main() {
	if err := cli.New().Run(context.TODO(), os.Args); err != nil {
		slog.Error("mugit", "err", err)
		os.Exit(1)
	}
}
