package gitx

import (
	"context"
	"fmt"
	"io"
)

// InfoRefs executes git-upload-pack --advertise-refs for smart-HTTP discovery.
func InfoRefs(ctx context.Context, repoDir string, out io.Writer) error {
	if err := PackLine(out, "# service=git-upload-pack\n"); err != nil {
		return fmt.Errorf("write pack line: %w", err)
	}

	if err := PackFlush(out); err != nil {
		return fmt.Errorf("flush pack: %w", err)
	}

	if err := gitCmd(ctx, cmdOpts{
		RepoDir: repoDir,
		Cmd: []string{
			"-c", "uploadpack.allowFilter=true",
			"upload-pack", "--stateless-rpc", "--advertise-refs",
		},
		Stdout: out,
		Stderr: out, // TODO: Check if this is correct.
	}); err != nil {
		return fmt.Errorf("git-upload-pack: %w", err)
	}
	return nil
}

// UploadPack executes git-upload-pack for smart-HTTP git fetch/clone.
// StatelessRPC should be true in case it's used over http, and false for ssh.
func UploadPack(ctx context.Context, repoDir string, statelessRPC bool, in io.Reader, out io.Writer) error {
	cmd := []string{"-c", "uploadpack.allowFilter=true", "upload-pack"}
	if statelessRPC {
		cmd = append(cmd, "--stateless-rpc")
	}

	if err := gitCmd(ctx, cmdOpts{
		RepoDir: repoDir,
		Cmd:     cmd,
		Stdin:   in,
		Stdout:  out,
		Stderr:  out, // TODO: Check if this is correct.
	}); err != nil {
		return fmt.Errorf("git-upload-pack: %w", err)
	}
	return nil
}

// ReceivePack executes git-receive-pack for git push.
func ReceivePack(ctx context.Context, repoDir string, in io.Reader, out, errout io.Writer) error {
	if err := gitCmd(ctx, cmdOpts{
		RepoDir: repoDir,
		Cmd:     []string{"receive-pack"},
		Stdin:   in,
		Stdout:  out,
		Stderr:  errout,
	}); err != nil {
		return fmt.Errorf("git-receive-pack: %w", err)
	}
	return nil
}
