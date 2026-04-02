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
	mugitgit "olexsmir.xyz/mugit/internal/git"
	"olexsmir.xyz/mugit/internal/handlers"
)

var (
	mugitBin   string
	httpPort   int
	repoDir    string
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

	repoDir = filepath.Join(tmpDir, "repos")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create repo dir: %v\n", err)
		return 1
	}

	if err := buildMugitBinary(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build binary: %v\n", err)
		return 1
	}

	port, err := findFreePort()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to find free port: %v\n", err)
		return 1
	}
	httpPort = port

	pubKey, err := os.ReadFile("testscript/testdata/test_key.pub")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read test key: %v\n", err)
		return 1
	}

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
			Dir:     repoDir,
			Readmes: []string{"README.md"},
			Masters: []string{"master", "main"},
		},
		SSH: config.SSHConfig{
			Enable: true,
			User:   "git",
			Keys:   []string{string(pubKey)},
		},
		Mirror: config.MirrorConfig{Enable: false},
		Cache: config.CacheConfig{
			HomePage: 0,
			Readme:   0,
			Diff:     0,
		},
	}

	configPath = filepath.Join(tmpDir, "config.yaml")
	if err := writeConfig(configPath, cfg); err != nil {
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
	testscript.Run(t, testscript.Params{
		Dir: "testscript",
		Setup: func(env *testscript.Env) error {
			env.Setenv("MUGIT_BIN", mugitBin)
			env.Setenv("MUGIT_CONFIG", configPath)
			env.Setenv("REPOS", repoDir)
			env.Setenv("MPORT", strconv.Itoa(httpPort))
			env.Setenv("MURL", fmt.Sprintf("http://127.0.0.1:%d", httpPort))
			return nil
		},
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"mkrepo":       cmdMkrepo,
			"mkfile":       cmdMkfile,
			"mugit":        cmdMugit,
			"mksshwrapper": cmdMksshwrapper,
		},
	})
}

func buildMugitBinary() error {
	tmpDir, err := os.MkdirTemp("", "mugit-bin-*")
	if err != nil {
		return err
	}
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
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("port %d not ready after %s", port, timeout)
}

func writeConfig(path string, cfg *config.Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func cmdMkrepo(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! mkrepo")
	}
	if len(args) != 1 {
		ts.Fatalf("usage: mkrepo <name>")
	}

	name := args[0]
	repoPath := filepath.Join(repoDir, mugitgit.ResolveName(name))
	if _, err := os.Stat(repoPath); err == nil {
		ts.Fatalf("repo %s already exists", name)
	}

	if err := mugitgit.Init(repoPath); err != nil {
		ts.Fatalf("init repo: %v", err)
	}
}

func cmdMkfile(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! mkfile")
	}
	if len(args) != 2 {
		ts.Fatalf("usage: mkfile <path> <content>")
	}
	path := args[0]
	content := args[1]
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		ts.Fatalf("mkfile: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		ts.Fatalf("mkfile: %v", err)
	}
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

func cmdMksshwrapper(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! mksshwrapper")
	}
	content := fmt.Sprintf("#!/bin/sh\nexport SSH_ORIGINAL_COMMAND=\"$2\"\nexec %s shell -c %s\n", mugitBin, configPath)
	if err := os.WriteFile(ts.Getenv("WORK")+"/ssh-wrapper.sh", []byte(content), 0o755); err != nil {
		ts.Fatalf("mksshwrapper: %v", err)
	}
}
