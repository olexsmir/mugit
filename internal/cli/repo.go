package cli

import (
	"context"
	"fmt"
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

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("repository already exists: %s", name)
	}

	if err := git.Init(path); err != nil {
		return err
	}

	mirrorURL := cmd.String("mirror")
	if mirrorURL != "" {
		if !strings.HasPrefix(mirrorURL, "http") {
			return fmt.Errorf("only http and https remotes are supported")
		}
		repo, err := git.Open(path, "")
		if err != nil {
			return fmt.Errorf("failed to open repo for mirror setup: %w", err)
		}
		if err := repo.SetMirrorRemote(mirrorURL); err != nil {
			return fmt.Errorf("failed to set mirror remote: %w", err)
		}
	}

	return nil
}
