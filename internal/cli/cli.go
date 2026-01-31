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
	"olexsmir.xyz/mugit/internal/config"
	"olexsmir.xyz/mugit/internal/handlers"
	"olexsmir.xyz/mugit/internal/mirror"
	"olexsmir.xyz/mugit/internal/ssh"
)

type Cli struct {
	cfg *config.Config
}

func New() *Cli {
	return &Cli{}
}

func (c *Cli) Run(ctx context.Context, args []string) error {
	cmd := &cli.Command{
		Name:                  "mugit",
		Usage:                 "a frontend for git repos",
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "path to config file",
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			loadedCfg, err := config.Load(cmd.String("config"))
			if err != nil {
				return ctx, err
			}
			c.cfg = loadedCfg
			return ctx, nil
		},
		Commands: []*cli.Command{
			{
				Name:   "serve",
				Usage:  "starts the server",
				Action: c.serveAction,
			},
			{
				Name: "repo",
				Commands: []*cli.Command{
					{
						Name:   "new",
						Usage:  "create new repo",
						Action: c.repoNewAction,
						Arguments: []cli.Argument{
							&cli.StringArg{Name: "name"},
						},
					},
				},
			},
		},
	}
	return cmd.Run(ctx, args)
}

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
