package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
	"olexsmir.xyz/mugit/internal/config"
	"olexsmir.xyz/mugit/internal/git"
	"olexsmir.xyz/mugit/internal/ssh"
)

type Cli struct {
	cfg     *config.Config
	ssh     *ssh.Shell
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
			loadedCfg, err := config.Load(
				config.PathOrDefault(cmd.String("config")))
			if err != nil {
				return ctx, err
			}
			c.cfg = loadedCfg

			if c.cfg.SSH.Enable {
				shell, err := ssh.NewShell(c.cfg)
				if err != nil {
					return ctx, err
				}
				c.ssh = shell
			}

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
							&cli.StringFlag{
								Name:    "description",
								Usage:   "set repo description",
								Aliases: []string{"desc"},
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
					{
						Name:   "set-default",
						Usage:  "switch repo's default branch",
						Action: c.repoDefaultAction,
						Arguments: []cli.Argument{
							&cli.StringArg{Name: "name"},
						},
					},
				},
			},
			{
				Name:        "shell",
				Description: "sshd things", // TODO: update me
				Action:      c.sshShellAction,
				Commands: []*cli.Command{
					{
						Name:   "keys",
						Action: c.sshAuthorizedKeysAction,
					},
				},
			},
		},
	}
	return cmd.Run(ctx, args)
}

func (c *Cli) openRepo(name string) (*git.Repo, error) {
	path, err := git.ResolvePath(c.cfg.Repo.Dir, name)
	if err != nil {
		return nil, err
	}

	repo, err := git.Open(path, "")
	if err != nil {
		return nil, fmt.Errorf("failed to open repo: %w", err)
	}

	return repo, nil
}
