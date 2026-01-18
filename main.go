package main

import (
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"

	"olexsmir.xyz/mugit/internal/config"
	"olexsmir.xyz/mugit/internal/handlers"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("main: %s", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load("/home/olex/code/mugit/config.yml")
	if err != nil {
		slog.Error("config error", "err", err)
		return err
	}

	mux := handlers.InitRoutes(cfg)

	port := strconv.Itoa(cfg.Server.Port)
	slog.Info("starting server", "host", cfg.Server.Host, "port", port)
	if err = http.ListenAndServe(net.JoinHostPort(cfg.Server.Host, port), mux); err != nil {
		slog.Error("server error", "err", err)
	}

	return nil
}
