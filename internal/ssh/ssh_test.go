package ssh

import (
	"strings"
	"testing"

	"olexsmir.xyz/mugit/internal/config"
	"olexsmir.xyz/x/is"
)

var validKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl"

func TestNewShell(t *testing.T) {
	tests := []struct {
		name    string
		keys    []string
		wantErr string
	}{
		{"valid key", []string{validKey}, ""},
		{"invalid key", []string{"invalid-key"}, "ssh: no key found"},
		{"multiple keys", []string{validKey, validKey}, ""},
		{"no keys", []string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{SSH: config.SSHConfig{Keys: tt.keys}}
			shell, err := NewShell(cfg)
			if tt.wantErr == "" {
				is.Err(t, err, nil)
				is.Equal(t, len(shell.keys), len(cfg.SSH.Keys))
			} else {
				is.Err(t, err, tt.wantErr)
			}
		})
	}
}

func TestShellParseCommand(t *testing.T) {
	cfg := &config.Config{
		SSH: config.SSHConfig{
			Keys: []string{validKey},
		},
	}

	shell, err := NewShell(cfg)
	is.Err(t, err, nil)

	tests := []struct {
		cmd        string
		wantGitCmd string
		wantRepo   string
		wantErr    string
	}{
		{"git-upload-pack 'myrepo'", "git-upload-pack", "myrepo", ""},
		{"git-upload-pack \"myrepo\"", "git-upload-pack", "myrepo", ""},
		{"git-upload-pack myrepo", "git-upload-pack", "myrepo", ""},
		{"git-upload-archive 'archive-repo'", "git-upload-archive", "archive-repo", ""},
		{"git-upload-pack", "", "", "invalid command"},
		{"git-upload-pack ''", "", "", "empty repository name"},
		{"git-receive-pack repo.git && echo hi", "", "", "invalid command"},
		{"echo hi", "", "", "invalid command"},
		{"", "", "", "invalid command"},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			gitCmd, repo, err := shell.parseCommand(tt.cmd)
			if tt.wantErr == "" {
				is.Err(t, err, nil)
				is.Equal(t, gitCmd, tt.wantGitCmd)
				is.Equal(t, repo, tt.wantRepo)
			} else {
				is.Err(t, err, tt.wantErr)
			}
		})
	}
}

func TestShellAuthorizedKeys(t *testing.T) {
	shell, err := NewShell(&config.Config{
		SSH: config.SSHConfig{Keys: []string{validKey}},
	})
	is.Err(t, err, nil)

	result := shell.AuthorizedKeys("/usr/bin/mugit")
	if !strings.Contains(result, `command="/usr/bin/mugit shell",no-port-forwarding,no-X11-forwarding,no-agent-forwarding,no-pty`) {
		t.Errorf("AuthorizedKeys() missing expected format\ngot: %s", result)
	}
	if !strings.Contains(result, validKey) {
		t.Errorf("AuthorizedKeys() missing SSH key\ngot: %s", result)
	}
}
