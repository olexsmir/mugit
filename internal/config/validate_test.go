package config

import (
	"testing"

	"olexsmir.xyz/x/is"
)

func TestCheckPort(t *testing.T) {
	is.Err(t, checkPort(1), nil)
	is.Err(t, checkPort(80), nil)
	is.Err(t, checkPort(65535), nil)

	is.Err(t, checkPort(0), "must be between")
	is.Err(t, checkPort(-1), "must be between")
	is.Err(t, checkPort(65536), "must be between")
}

func TestConfig_Validate(t *testing.T) {
	hostKey := "testdata/hostkey"
	tests := []struct {
		name     string
		expected any
		c        Config
	}{
		{
			name: "minimal",
			c: Config{
				Meta: MetaConfig{Host: "example.com"},
				Repo: RepoConfig{Dir: t.TempDir()},
			},
		},
		{
			name: "minimal with ssh",
			c: Config{
				Meta: MetaConfig{Host: "example.com"},
				Repo: RepoConfig{Dir: t.TempDir()},
				SSH: SSHConfig{
					Enable:  true,
					HostKey: hostKey,
				},
			},
		},
		{
			name:     "not set meta.host",
			expected: "meta.host is required",
			c: Config{
				Repo: RepoConfig{Dir: t.TempDir()},
			},
		},
		{
			name:     "invalid meta.host",
			expected: "meta.host shouldn't include protocol",
			c: Config{
				Meta: MetaConfig{Host: "https://example.com"},
				Repo: RepoConfig{Dir: t.TempDir()},
			},
		},
		{
			name:     "invalid repo.dir",
			expected: "repo.dir",
			c: Config{
				Meta: MetaConfig{Host: "example.com"},
				Repo: RepoConfig{Dir: "nonexistent"},
			},
		},
		{
			name:     "invalid server port",
			expected: "server.port",
			c: Config{
				Meta:   MetaConfig{Host: "example.com"},
				Repo:   RepoConfig{Dir: t.TempDir()},
				Server: ServerConfig{Port: -1},
			},
		},
		{
			name:     "invalid ssh port",
			expected: "ssh.port",
			c: Config{
				Meta: MetaConfig{Host: "example.com"},
				Repo: RepoConfig{Dir: t.TempDir()},
				SSH: SSHConfig{
					Enable:  true,
					HostKey: hostKey,
					Port:    100000,
				},
			},
		},
		{
			name:     "same ssh and http ports",
			expected: "ssh.port must differ",
			c: Config{
				Meta:   MetaConfig{Host: "example.com"},
				Repo:   RepoConfig{Dir: t.TempDir()},
				SSH:    SSHConfig{Enable: true, Port: 228},
				Server: ServerConfig{Port: 228},
			},
		},
		{
			name:     "invalid ssh.host_key path",
			expected: "ssh.host_key",
			c: Config{
				Meta: MetaConfig{Host: "example.com"},
				Repo: RepoConfig{Dir: t.TempDir()},
				SSH: SSHConfig{
					Enable:  true,
					HostKey: "/somewhere",
				},
			},
		},
		{
			name:     "invalid mirror.interval duration format",
			expected: "mirror.interval: invalid duration",
			c: Config{
				Meta: MetaConfig{Host: "example.com"},
				Repo: RepoConfig{Dir: t.TempDir()},
				Mirror: MirrorConfig{
					Enable:   true,
					Interval: "asdf",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.c.ensureDefaults()
			err := tt.c.validate()
			is.Err(t, err, tt.expected)
		})
	}
}
