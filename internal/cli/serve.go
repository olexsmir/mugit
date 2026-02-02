package cli

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/urfave/cli/v3"
	"olexsmir.xyz/mugit/internal/handlers"
	"olexsmir.xyz/mugit/internal/mirror"
	"olexsmir.xyz/mugit/internal/ssh"
)

func (c *Cli) serveAction(ctx context.Context, cmd *cli.Command) error {
	httpServer := &http.Server{
		Addr:    net.JoinHostPort(c.cfg.Server.Host, strconv.Itoa(c.cfg.Server.Port)),
		Handler: handlers.InitRoutes(c.cfg),
	}
	go func() {
		slog.Info("starting http server", "host", c.cfg.Server.Host, "port", c.cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "err", err)
		}
	}()

	if c.cfg.SSH.Enable {
		sshServer := ssh.NewServer(c.cfg)
		go func() {
			slog.Info("starting ssh server", "port", c.cfg.SSH.Port)
			if err := sshServer.Start(); err != nil {
				slog.Error("ssh server error", "err", err)
			}
		}()
	}

	if c.cfg.Mirror.Enable {
		mirrorer := mirror.NewWorker(c.cfg)
		go func() {
			slog.Info("starting mirroring worker")
			mirrorer.Start(context.TODO())
		}()
	}

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
