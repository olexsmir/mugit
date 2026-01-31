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

	// TODO: check if there's already such repo

	path, err := securejoin.SecureJoin(c.cfg.Repo.Dir, name)
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("repository already exists: %s", name)
	}

	return git.Init(path)
}
