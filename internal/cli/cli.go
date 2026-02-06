package cli

import (
	"context"
	"fmt"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/urfave/cli/v3"
	"olexsmir.xyz/mugit/internal/config"
	"olexsmir.xyz/mugit/internal/git"
)

type Cli struct {
	cfg     *config.Config
	version string
}

func New(version string) *Cli {
	return &Cli{
		version: version,
	}
}

func (c *Cli) Run(ctx context.Context, args []string) error {
	cmd := &cli.Command{
		Name:                  "mugit",
		Usage:                 "a frontend for git repos",
		Version:               c.version,
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
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "mirror",
								Usage: "remote URL(only http/https) to mirror from",
							},
							&cli.BoolFlag{
								Name:  "private",
								Usage: "make the repository private",
							},
						},
					},
					{
						Name:   "description",
						Usage:  "get or set repo description",
						Action: c.repoDescriptionAction,
						Arguments: []cli.Argument{
							&cli.StringArg{Name: "name"},
						},
					},
					{
						Name:   "private",
						Usage:  "toggle private status of a repo",
						Action: c.repoPrivateAction,
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

func (c *Cli) openRepo(name string) (*git.Repo, error) {
	path, err := securejoin.SecureJoin(c.cfg.Repo.Dir, name)
	if err != nil {
		return nil, err
	}

	repo, err := git.Open(path, "")
	if err != nil {
		return nil, fmt.Errorf("failed to open repo: %w", err)
	}

	return repo, nil
}
