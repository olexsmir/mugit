package gitservice

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
)

// InfoRefs executes git-upload-pack --advertise-refs for smart-HTTP discovery.
func InfoRefs(dir string, out io.Writer) error {
	cmd := exec.Command("git",
		"upload-pack",
		"--stateless-rpc",
		"--advertise-refs",
		".")
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout

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
	if _, err := io.Copy(&buf, stdout); err != nil {
		return fmt.Errorf("copy stdout to buffer: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("git-upload-pack: %w", err)
	}

	if _, err := io.Copy(out, &buf); err != nil {
		return fmt.Errorf("copy buffer to output: %w", err)
	}

	return nil
}

// UploadPack executes git-upload-pack for smart-HTTP git fetch/clone.
// StatelessRPC should be true in case it's used over http, and false for ssh.
func UploadPack(dir string, statelessRPC bool, in io.Reader, out io.Writer) error {
	return gitCmd("upload-pack", config{
		Dir:          dir,
		StatelessRPC: statelessRPC,
		AllowFilter:  true,
		Stdin:        in,
		Stdout:       out,
	})
}

func ReceivePack(dir string, in io.Reader, out io.Writer) error {
	return gitCmd("receive-pack", config{
		Dir:    dir,
		Stdin:  in,
		Stdout: out,
	})
}

type config struct {
	Dir          string
	StatelessRPC bool
	AllowFilter  bool
	ExtraArgs    []string
	Stdin        io.Reader
	Stdout       io.Writer
}

func gitCmd(service string, c config) error {
	args := []string{}
	if c.AllowFilter {
		args = append(args, "-c", "uploadpack.allowFilter=true")
	}

	args = append(args, service)
	if c.StatelessRPC {
		args = append(args, "--stateless-rpc")
	}

	args = append(args, c.ExtraArgs...)
	args = append(args, ".")

	cmd := exec.Command("git", args...)
	cmd.Dir = c.Dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var (
		err   error
		stdin io.WriteCloser
	)

	if c.Stdin != nil {
		stdin, err = cmd.StdinPipe()
		if err != nil {
			return err
		}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", service, err)
	}

	if c.Stdin != nil {
		// Don't add to WaitGroup - stdin closes when client closes,
		// shouldn't block waiting for output to finish
		go func() {
			defer stdin.Close()
			io.Copy(stdin, c.Stdin)
		}()
	}

	var wg sync.WaitGroup
	var stdoutErr error

	wg.Go(func() {
		_, stdoutErr = io.Copy(c.Stdout, stdout)
	})

	wg.Wait()

	if stdoutErr != nil {
		return fmt.Errorf("copy stdout: %w", stdoutErr)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%s: %w", service, err)
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
