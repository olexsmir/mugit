package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

var errSSHDisabled = errors.New("ssh is disabled")

func (c *Cli) sshShellAction(ctx context.Context, cmd *cli.Command) error {
	if !c.cfg.SSH.Enable {
		return errSSHDisabled
	}

	sshCommand := os.Getenv("SSH_ORIGINAL_COMMAND")
	if err := c.ssh.HandleCommand(ctx, sshCommand, os.Stdin, os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
		return nil
	}
	return nil
}

func (c *Cli) sshAuthorizedKeysAction(ctx context.Context, cmd *cli.Command) error {
	if !c.cfg.SSH.Enable {
		return errSSHDisabled
	}

	fingerprint := cmd.Args().First()
	if fingerprint == "" {
		return fmt.Errorf("fingerprint is required")
	}

	executablePath, err := os.Executable()
	if err != nil {
		return err
	}

	out := c.ssh.AuthorizedKeys(executablePath)
	fmt.Fprint(os.Stdout, out)

	return nil
}
