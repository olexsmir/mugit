package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/urfave/cli/v3"
	"olexsmir.xyz/mugit/internal/git"
)

func (c *Cli) repoNewAction(ctx context.Context, cmd *cli.Command) error {
	name := cmd.StringArg("name")
	if name == "" {
		return fmt.Errorf("no name provided")
	}

	name = strings.TrimRight(name, ".git") + ".git"

	path, err := securejoin.SecureJoin(c.cfg.Repo.Dir, name)
	if err != nil {
		return err
	}

	if _, err = os.Stat(path); err == nil {
		return fmt.Errorf("repository already exists: %s", name)
	}

	if err = git.Init(path); err != nil {
		return err
	}

	repo, err := git.Open(path, "")
	if err != nil {
		return fmt.Errorf("failed to open repo: %w", err)
	}

	if err := repo.SetPrivate(cmd.Bool("private")); err != nil {
		return fmt.Errorf("failed to set private status: %w", err)
	}

	mirrorURL := cmd.String("mirror")
	if mirrorURL != "" {
		if !strings.HasPrefix(mirrorURL, "http") {
			return fmt.Errorf("only http and https remotes are supported")
		}
		if err := repo.SetMirrorRemote(mirrorURL); err != nil {
			return fmt.Errorf("failed to set mirror remote: %w", err)
		}
	}

	return nil
}

func (c *Cli) repoDescriptionAction(ctx context.Context, cmd *cli.Command) error {
	name := cmd.StringArg("name")
	if name == "" {
		return fmt.Errorf("no name provided")
	}

	name = strings.TrimRight(name, ".git") + ".git"

	repo, err := c.openRepo(name)
	if err != nil {
		return fmt.Errorf("failed to open repo: %w", err)
	}

	newDesc := cmd.Args().Get(0)
	if newDesc != "" {
		if err = repo.SetDescription(newDesc); err != nil {
			return fmt.Errorf("failed to set description: %w", err)
		}
	}

	desc, err := repo.Description()
	if err != nil {
		return fmt.Errorf("failed to get description: %w", err)
	}

	slog.Info("changed repo description", "repo", name, "new_description", desc)
	return nil
}

func (c *Cli) repoPrivateAction(ctx context.Context, cmd *cli.Command) error {
	name := cmd.StringArg("name")
	if name == "" {
		return fmt.Errorf("no name provided")
	}

	name = strings.TrimRight(name, ".git") + ".git"

	repo, err := c.openRepo(name)
	if err != nil {
		return fmt.Errorf("failed to open repo: %w", err)
	}

	isPrivate, err := repo.IsPrivate()
	if err != nil {
		return fmt.Errorf("failed to get private status: %w", err)
	}

	newStatus := !isPrivate
	if err := repo.SetPrivate(newStatus); err != nil {
		return fmt.Errorf("failed to set private status: %w", err)
	}

	slog.Info("new repo private status", "repo", name, "is_private", newStatus)
	return nil
}
