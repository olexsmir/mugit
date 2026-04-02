package main_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/rogpeppe/go-internal/testscript"
	"gopkg.in/yaml.v2"

	"olexsmir.xyz/mugit/internal/config"
	"olexsmir.xyz/mugit/internal/handlers"
)

var (
	mugitBin   string
	httpPort   int
	reposDir   string
	configPath string
)

func TestMain(m *testing.M) { os.Exit(testMain(m)) }
func testMain(m *testing.M) int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "mugit-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		return 1
	}
	defer os.RemoveAll(tmpDir)

	reposDir = filepath.Join(tmpDir, "repos")
	if jerr := os.MkdirAll(reposDir, 0o755); jerr != nil {
		fmt.Fprintf(os.Stderr, "failed to create repo dir: %v\n", jerr)
		return 1
	}

	if berr := buildMugitBinary(tmpDir); berr != nil {
		fmt.Fprintf(os.Stderr, "failed to build binary: %v\n", berr)
		return 1
	}

	port, err := findFreePort()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to find free port: %v\n", err)
		return 1
	}
	httpPort = port

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "127.0.0.1",
			Port: httpPort,
		},
		Meta: config.MetaConfig{
			Title: "test mugit",
			Host:  "localhost",
		},
		Repo: config.RepoConfig{
			Dir:     reposDir,
			Readmes: []string{"README.md"},
			Masters: []string{"master", "main"},
		},
		SSH:    config.SSHConfig{Enable: true, User: "git"},
		Mirror: config.MirrorConfig{Enable: false},
		Cache: config.CacheConfig{
			HomePage: 0,
			Readme:   0,
			Diff:     0,
		},
	}

	configPath = filepath.Join(tmpDir, "config.yaml")
	configBytes, err := yaml.Marshal(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal config: %v\n", err)
		return 1
	}
	if err := os.WriteFile(configPath, configBytes, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write config: %v\n", err)
		return 1
	}

	httpServer := &http.Server{
		Addr:    net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port)),
		Handler: handlers.InitRoutes(cfg),
	}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
		}
	}()

	if err := waitForPort(httpPort, 5*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "server did not become ready: %v\n", err)
		return 1
	}

	code := m.Run()
	httpServer.Shutdown(ctx)
	return code
}

func TestScript(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests")
	}

	sshWrapperContent := fmt.Sprintf(`#!/bin/sh
export SSH_ORIGINAL_COMMAND="$2"
exec %s shell -c %s`, mugitBin, configPath)

	testscript.Run(t, testscript.Params{
		Dir: "testscript",
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"mugit": cmdMugit,
			"git":   cmdGit,
		},
		Setup: func(env *testscript.Env) error {
			work := env.Getenv("WORK")
			sshWrapperPath := filepath.Join(work, "ssh-wrapper.sh")
			if err := os.WriteFile(sshWrapperPath, []byte(sshWrapperContent), 0o700); err != nil {
				return fmt.Errorf("failed to create ssh wrapper: %w", err)
			}

			env.Setenv("SSH_WRAPPER", sshWrapperPath)
			env.Setenv("REPOS", reposDir)
			env.Setenv("MPORT", strconv.Itoa(httpPort))
			env.Setenv("MURL", fmt.Sprintf("http://127.0.0.1:%d", httpPort))
			return nil
		},
	})
}

func buildMugitBinary(tmpDir string) error {
	mugitBin = filepath.Join(tmpDir, "mugit")
	cmd := exec.Command("go", "build", "-o", mugitBin, ".")
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir)
		return fmt.Errorf("go build: %v\n%s", err, out)
	}
	return nil
}

func findFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}

func waitForPort(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if conn, err := net.DialTimeout(
			"tcp",
			net.JoinHostPort("127.0.0.1", strconv.Itoa(port)),
			200*time.Millisecond,
		); err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("port %d not ready after %s", port, timeout)
}

func cmdMugit(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) < 1 {
		ts.Fatalf("usage: mugit <subcommand> ...")
	}
	cmd := exec.Command(mugitBin, append([]string{"-c", configPath}, args...)...)
	cmd.Env = os.Environ()
	cmd.Stdout = ts.Stdout()
	cmd.Stderr = ts.Stderr()
	err := cmd.Run()
	if neg {
		if err == nil {
			ts.Fatalf("expected mugit to fail, it succeeded")
		}
	} else {
		if err != nil {
			ts.Fatalf("mugit: %v", err)
		}
	}
}

func cmdGit(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) > 0 && args[0] == "init" {
		hasBranch := false
		for _, arg := range args {
			if arg == "-b" || arg == "--initial-branch" {
				hasBranch = true
				break
			}
		}
		if !hasBranch {
			args = append([]string{"init", "-b", "master"}, args[1:]...)
		}
	}
	args = append([]string{
		"-c", "user.email=test@test.local",
		"-c", "user.name=Test User",
	}, args...)
	cmd := exec.Command("git", args...)
	cmd.Dir = ts.Getenv("WORK")
	cmd.Stdout = ts.Stdout()
	cmd.Stderr = ts.Stderr()

	err := cmd.Run()
	if err == nil && neg {
		ts.Fatalf("expected git to fail, but it succeeded")
	}
	if err != nil && !neg {
		ts.Fatalf("git: %v", err)
	}
}
