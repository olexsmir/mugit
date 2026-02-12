package gitx

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"syscall"
)

// InfoRefs executes git-upload-pack --advertise-refs for smart-HTTP discovery.
func InfoRefs(ctx context.Context, repoDir string, out io.Writer) error {
	cmd := exec.CommandContext(ctx, "git", []string{
		"-c", "uploadpack.allowFilter=true",
		"upload-pack",
		"--stateless-rpc",
		"--advertise-refs",
		".",
	}...)
	cmd.Dir = repoDir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = gitEnv

	cmd.Stdout = out
	cmd.Stderr = out // TODO: Check if this is correct.

	if err := PackLine(out, "# service=git-upload-pack\n"); err != nil {
		return fmt.Errorf("write pack line: %w", err)
	}
	if err := PackFlush(out); err != nil {
		return fmt.Errorf("flush pack: %w", err)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git-upload-pack: %w", err)
	}

	return nil
}

// UploadPack executes git-upload-pack for smart-HTTP git fetch/clone.
// StatelessRPC should be true in case it's used over http, and false for ssh.
func UploadPack(ctx context.Context, repoDir string, statelessRPC bool, in io.Reader, out io.Writer) error {
	args := []string{"-c", "uploadpack.allowFilter=true", "upload-pack"}
	if statelessRPC {
		args = append(args, "--stateless-rpc")
	}
	args = append(args, ".")

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoDir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = gitEnv

	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = out // TODO: Check if this is correct.

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git-upload-pack: %w", err)
	}

	return nil
}

// ReceivePack executes git-receive-pack for git push.
func ReceivePack(ctx context.Context, repoDir string, in io.Reader, out, errout io.Writer) error {
	cmd := exec.CommandContext(ctx, "git", "receive-pack", ".")
	cmd.Dir = repoDir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = gitEnv

	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errout

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git-receive-pack: %w", err)
	}

	return nil
}
