package git

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"
)

var gitEnv = []string{
	"GIT_CONFIG_GLOBAL=/dev/null",
	"GIT_CONFIG_SYSTEM=/dev/null",
}

type cmdOpts struct {
	Cmd         []string
	GitProtocol string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
}

func (g *Repo) gitCmd(ctx context.Context, opts cmdOpts) error {
	opts.Cmd = append(opts.Cmd, ".")
	cmd := exec.CommandContext(ctx, "git", opts.Cmd...)
	cmd.Dir = g.path
	cmd.Env = append(gitEnv, fmt.Sprintf("GIT_PROTOCOL=%s", opts.GitProtocol))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdin = cmp.Or[io.Reader](opts.Stdin, strings.NewReader(""))
	cmd.Stdout = cmp.Or(opts.Stdout, io.Discard)
	cmd.Stderr = cmp.Or(opts.Stderr, io.Discard)
	return cmd.Run()
}

func (g *Repo) streamingGitLog(ctx context.Context, extraArgs ...string) (io.ReadCloser, error) {
	args := []string{"log", g.h.String()}
	args = append(args, extraArgs...)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.path

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &processReader{
		Reader: stdout,
		cmd:    cmd,
		stdout: stdout,
	}, nil
}

// processReader wraps a reader and ensures the associated process is cleaned up
type processReader struct {
	io.Reader
	cmd    *exec.Cmd
	stdout io.ReadCloser
}

func (pr *processReader) Close() error {
	if err := pr.stdout.Close(); err != nil {
		return err
	}
	return pr.cmd.Wait()
}
