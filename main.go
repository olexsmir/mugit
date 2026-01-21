package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"olexsmir.xyz/mugit/internal/config"
	"olexsmir.xyz/mugit/internal/handlers"
	"olexsmir.xyz/mugit/internal/mirror"
	"olexsmir.xyz/mugit/internal/ssh"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("main: %s", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load("/home/olex/mugit-test/config.yml")
	if err != nil {
		slog.Error("config error", "err", err)
		return err
	}

	httpServer := &http.Server{
		Addr:    net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port)),
		Handler: handlers.InitRoutes(cfg),
	}

	go func() {
		slog.Info("starting http server", "host", cfg.Server.Host, "port", cfg.Server.Port)

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "err", err)
		}
	}()

	sshServer := ssh.NewServer(cfg)
	if cfg.SSH.Enable {
		go func() {
			slog.Info("starting ssh server", "port", cfg.SSH.Port)
			if err := sshServer.Start(); err != nil {
				slog.Error("ssh server error", "err", err)
			}
		}()
	}

	mirrorer := mirror.NewWorker(cfg)
	if cfg.Mirror.Enable {
		go func() {
			slog.Info("starting mirroring worker")
			mirrorer.Start(context.TODO())
		}()
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	slog.Info("received signal, starting graceful shutdown", "signal", sig)

	if err := httpServer.Shutdown(context.TODO()); err != nil {
		slog.Error("HTTP server shutdown error", "err", err)
	} else {
		slog.Info("HTTP server shutdown complete")
	}

	return nil
}
