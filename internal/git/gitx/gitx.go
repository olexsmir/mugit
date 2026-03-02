package gitx

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"syscall"
)

var gitEnv = []string{
	"GIT_CONFIG_GLOBAL=/dev/null",
	"GIT_CONFIG_SYSTEM=/dev/null",
}

type cmdOpts struct {
	Cmd         []string
	GitProtocol string
	RepoDir     string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
}

func gitCmd(ctx context.Context, opts cmdOpts) error {
	opts.Cmd = append(opts.Cmd, ".")
	cmd := exec.CommandContext(ctx, "git", opts.Cmd...)
	cmd.Dir = opts.RepoDir
	cmd.Env = append(gitEnv, fmt.Sprintf("GIT_PROTOCOL=%s", opts.GitProtocol))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdin = opts.Stdin
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr
	return cmd.Run()
}
