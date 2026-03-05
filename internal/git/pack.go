package git

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// InfoRefs executes git-upload-pack --advertise-refs for smart-HTTP discovery.
func (g *Repo) InfoRefs(ctx context.Context, protocol string, out io.Writer) error {
	if !strings.Contains(protocol, "version=2") {
		if err := PackLine(out, "# service=git-upload-pack\n"); err != nil {
			return fmt.Errorf("write pack line: %w", err)
		}
		if err := PackFlush(out); err != nil {
			return fmt.Errorf("flush pack: %w", err)
		}
	}

	if err := gitCmd(ctx, cmdOpts{
		GitProtocol: protocol,
		Cmd: []string{
			"-c", "uploadpack.allowFilter=true",
			"upload-pack", "--stateless-rpc", "--advertise-refs",
		},
		RepoDir: g.path,
		Stdout:  out,
		Stderr:  out, // TODO: Check if this is correct.
	}); err != nil {
		return fmt.Errorf("git-upload-pack: %w", err)
	}
	return nil
}

// UploadPack executes git-upload-pack for smart-HTTP git fetch/clone.
// StatelessRPC should be true in case it's used over http, and false for ssh.
func (g *Repo) UploadPack(ctx context.Context, statelessRPC bool, protocol string, in io.Reader, out io.Writer) error {
	cmd := []string{"-c", "uploadpack.allowFilter=true", "upload-pack"}
	if statelessRPC {
		cmd = append(cmd, "--stateless-rpc")
	}

	if err := gitCmd(ctx, cmdOpts{
		Cmd:         cmd,
		GitProtocol: protocol,
		RepoDir:     g.path,
		Stdin:       in,
		Stdout:      out,
		Stderr:      out, // TODO: Check if this is correct.
	}); err != nil {
		return fmt.Errorf("git-upload-pack: %w", err)
	}
	return nil
}

// ReceivePack executes git-receive-pack for git push.
func (g *Repo) ReceivePack(ctx context.Context, in io.Reader, out, errout io.Writer) error {
	if err := gitCmd(ctx, cmdOpts{
		RepoDir: g.path,
		Cmd:     []string{"receive-pack"},
		Stdin:   in,
		Stdout:  out,
		Stderr:  errout,
	}); err != nil {
		return fmt.Errorf("git-receive-pack: %w", err)
	}
	return nil
}

// PackLine writes a pkt-line formatted string.
func PackLine(w io.Writer, s string) error {
	_, err := fmt.Fprintf(w, "%04x%s", len(s)+4, s)
	return err
}

// PackFlush writes a flush packet.
func PackFlush(w io.Writer) error {
	_, err := fmt.Fprint(w, "0000")
	return err
}

// PackError writes an ERR packet for protocol-level errors.
// Git displays this as: fatal: remote error: <msg>
func PackError(w io.Writer, msg string) error {
	return PackLine(w, "ERR "+msg)
}
