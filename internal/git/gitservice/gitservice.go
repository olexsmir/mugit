package gitservice

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"
)

// Thanks https://git.icyphox.sh/legit/blob/master/git/service/service.go

// InfoRefs executes git-upload-pack --advertise-refs for smart-HTTP discovery.
func InfoRefs(dir string, out io.Writer) error {
	cmd := exec.Command("git",
		"upload-pack",
		"--stateless-rpc",
		"--advertise-refs",
		".")
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdoutPipe, _ := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout
	defer stdoutPipe.Close()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start git-upload-pack: %w", err)
	}

	if err := packLine(out, "# service=git-upload-pack\n"); err != nil {
		return fmt.Errorf("write pack line: %w", err)
	}

	if err := packFlush(out); err != nil {
		return fmt.Errorf("flush pack: %w", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, stdoutPipe); err != nil {
		return fmt.Errorf("copy stdout to buffer: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		var out strings.Builder
		io.Copy(&out, &buf)
		return fmt.Errorf("git-upload-pack: %w: %s", err, out.String())
	}

	if _, err := io.Copy(out, &buf); err != nil {
		return fmt.Errorf("copy buffer to output: %w", err)
	}

	return nil
}

// UploadPack executes git-upload-pack for smart-HTTP git fetch/clone.
func UploadPack(dir string, in io.Reader, out io.Writer) error {
	cmd := exec.Command("git",
		"-c", "uploadpack.allowFilter=true",
		"upload-pack",
		"--stateless-rpc",
		".")
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdoutPipe, _ := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout
	defer stdoutPipe.Close()

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	defer stdinPipe.Close()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start git-upload-pack: %w", err)
	}

	if _, err := io.Copy(stdinPipe, in); err != nil {
		return fmt.Errorf("copy to stdin: %w", err)
	}
	stdinPipe.Close()

	if _, err := io.Copy(out, stdoutPipe); err != nil {
		return fmt.Errorf("copy stdout: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("git-upload-pack: %w", err)
	}

	return nil
}

func packLine(w io.Writer, s string) error {
	_, err := fmt.Fprintf(w, "%04x%s", len(s)+4, s)
	return err
}

func packFlush(w io.Writer) error {
	_, err := fmt.Fprint(w, "0000")
	return err
}
