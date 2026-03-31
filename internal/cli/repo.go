package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/urfave/cli/v3"
	"olexsmir.xyz/mugit/internal/git"
	"olexsmir.xyz/mugit/internal/mirror"
)

func (c *Cli) repoNewAction(ctx context.Context, cmd *cli.Command) error {
	name, err := c.getRepoNameArg(cmd)
	if name == "" {
		return err
	}

	path, err := git.ResolvePath(c.cfg.Repo.Dir, name)
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
		if err := mirror.IsRemoteSupported(mirrorURL); err != nil {
			return err
		}
		if err := repo.SetMirrorRemote(mirrorURL); err != nil {
			return fmt.Errorf("failed to set mirror remote: %w", err)
		}
	}

	desc := cmd.String("description")
	if desc != "" {
		if err := repo.SetDescription(desc); err != nil {
			return fmt.Errorf("failed to set description: %w", err)
		}
	}

	return nil
}

func (c *Cli) repoDescriptionAction(ctx context.Context, cmd *cli.Command) error {
	name, err := c.getRepoNameArg(cmd)
	if name == "" {
		return err
	}

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
	name, err := c.getRepoNameArg(cmd)
	if name == "" {
		return err
	}

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

func (c *Cli) repoSetHeadAction(ctx context.Context, cmd *cli.Command) error {
	name, err := c.getRepoNameArg(cmd)
	if name == "" {
		return err
	}

	repo, err := c.openRepo(name)
	if err != nil {
		return fmt.Errorf("failed to open repo: %w", err)
	}

	branch := cmd.Args().Get(0)
	slog.Info("chaining repo head", "repo", name, "branch", branch)
	err = repo.Checkout(branch)
	return err
}

func (c *Cli) getRepoNameArg(cmd *cli.Command) (string, error) {
	name := cmd.StringArg("name")
	if name == "" {
		return "", fmt.Errorf("no name provided")
	}
	return git.ResolveName(name), nil
}
