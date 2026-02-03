package config

import (
	"os"
	"path/filepath"
	"testing"

	"olexsmir.xyz/x/is"
)

func TestFindConfigFile(t *testing.T) {
	t.Run("returns user provided path when it exists", func(t *testing.T) {
		path, err := findConfigFile("testdata/hostkey")
		is.Err(t, err, nil)
		is.Equal(t, path, "testdata/hostkey")
	})

	t.Run("falls back when user path doesn't exist", func(t *testing.T) {
		path, err := findConfigFile("/nonexistent/user/config.yaml")
		if err != nil {
			is.Err(t, err, ErrConfigNotFound)
		} else {
			_, statErr := os.Stat(path)
			is.Err(t, statErr, nil)
		}
	})

	t.Run("finds config in user config directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := filepath.Join(tmpDir, "mugit")
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			t.Fatal(err)
		}
		configFile := filepath.Join(configDir, "config.yaml")
		if err := os.WriteFile(configFile, []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}

		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		path, err := findConfigFile("")
		is.Err(t, err, nil)
		is.Equal(t, path, configFile)
	})

	t.Run("returns error when no config found anywhere", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/nonexistent")
		t.Setenv("HOME", "/nonexistent")

		path, err := findConfigFile("/nonexistent/config.yaml")
		is.Err(t, err, ErrConfigNotFound)
		is.Equal(t, path, "")
	})

	t.Run("prefers data directory over user config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := filepath.Join(tmpDir, "mugit")
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			t.Fatal(err)
		}
		userConfigFile := filepath.Join(configDir, "config.yaml")
		if err := os.WriteFile(userConfigFile, []byte("user config"), 0o644); err != nil {
			t.Fatal(err)
		}

		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		path, err := findConfigFile("")
		is.Err(t, err, nil)

		if path == "/var/lib/mugit/config.yaml" {
			_, statErr := os.Stat(path)
			is.Err(t, statErr, nil)
		} else {
			is.Equal(t, path, userConfigFile)
		}
	})
}

func TestValidatePort(t *testing.T) {
	t.Run("accepts standard port numbers", func(t *testing.T) {
		is.Err(t, validatePort(1, "test"), nil)
		is.Err(t, validatePort(80, "test"), nil)
		is.Err(t, validatePort(8080, "test"), nil)
		is.Err(t, validatePort(65535, "test"), nil)
	})

	t.Run("rejects out of range ports", func(t *testing.T) {
		is.Err(t, validatePort(0, "test"), "must be between")
		is.Err(t, validatePort(-1, "test"), "must be between")
		is.Err(t, validatePort(65536, "test"), "must be between")
		is.Err(t, validatePort(100000, "test"), "must be between")
	})
}

func TestValidateDirExists(t *testing.T) {
	t.Run("accepts existing directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		is.Err(t, validateDirExists(tmpDir, "test"), nil)
	})

	t.Run("rejects nonexistent paths", func(t *testing.T) {
		is.Err(t, validateDirExists("/nonexistent/path/to/dir", "test"), "does not exist")
	})

	t.Run("rejects empty paths", func(t *testing.T) {
		is.Err(t, validateDirExists("", "test"), "is required")
	})

	t.Run("rejects files when directory expected", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "file.txt")
		if err := os.WriteFile(tmpFile, []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
		is.Err(t, validateDirExists(tmpFile, "test"), "not a directory")
	})
}

func TestValidateFileExists(t *testing.T) {
	t.Run("accepts existing files", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "file.txt")
		if err := os.WriteFile(tmpFile, []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
		is.Err(t, validateFileExists(tmpFile, "test"), nil)
	})

	t.Run("rejects nonexistent files", func(t *testing.T) {
		is.Err(t, validateFileExists("/nonexistent/file.txt", "test"), "does not exist")
	})

	t.Run("rejects empty paths", func(t *testing.T) {
		is.Err(t, validateFileExists("", "test"), "is required")
	})

	t.Run("rejects directories when file expected", func(t *testing.T) {
		tmpDir := t.TempDir()
		is.Err(t, validateFileExists(tmpDir, "test"), "is a directory")
	})
}

func TestConfigValidate(t *testing.T) {
	tmpDir := t.TempDir()
	hostKeyPath := "testdata/hostkey"

	t.Run("accepts minimal valid configuration", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Host: "localhost",
				Port: 8080,
			},
			Meta: MetaConfig{
				Title:       "Test",
				Description: "Test description",
				Host:        "example.com",
			},
			Repo: RepoConfig{
				Dir:     tmpDir,
				Readmes: []string{"README.md"},
				Masters: []string{"main"},
			},
			SSH: SSHConfig{
				Enable: false,
			},
			Mirror: MirrorConfig{
				Enable: false,
			},
		}
		is.Err(t, cfg.validate(), nil)
	})

	t.Run("accepts configuration with SSH enabled", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Host: "localhost",
				Port: 8080,
			},
			Meta: MetaConfig{
				Title:       "Test",
				Description: "Test description",
				Host:        "example.com",
			},
			Repo: RepoConfig{
				Dir:     tmpDir,
				Readmes: []string{"README.md"},
				Masters: []string{"main"},
			},
			SSH: SSHConfig{
				Enable:  true,
				Port:    2222,
				HostKey: hostKeyPath,
				Keys:    []string{"ssh-rsa AAAAB3..."},
			},
			Mirror: MirrorConfig{
				Enable: false,
			},
		}
		is.Err(t, cfg.validate(), nil)
	})

	t.Run("accepts configuration with mirroring enabled", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Host: "localhost",
				Port: 8080,
			},
			Meta: MetaConfig{
				Title:       "Test",
				Description: "Test description",
				Host:        "example.com",
			},
			Repo: RepoConfig{
				Dir:     tmpDir,
				Readmes: []string{"README.md"},
				Masters: []string{"main"},
			},
			SSH: SSHConfig{
				Enable: false,
			},
			Mirror: MirrorConfig{
				Enable:      true,
				Interval:    "1h",
				GithubToken: "ghp_token",
			},
		}
		is.Err(t, cfg.validate(), nil)
	})

	t.Run("rejects invalid server port", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port: 0,
			},
			Meta: MetaConfig{
				Host: "example.com",
			},
			Repo: RepoConfig{
				Dir:     tmpDir,
				Readmes: []string{"README.md"},
				Masters: []string{"main"},
			},
		}
		is.Err(t, cfg.validate(), "server.port")
	})

	t.Run("rejects missing meta host", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port: 8080,
			},
			Meta: MetaConfig{
				Host: "",
			},
			Repo: RepoConfig{
				Dir:     tmpDir,
				Readmes: []string{"README.md"},
				Masters: []string{"main"},
			},
		}
		is.Err(t, cfg.validate(), "meta.host")
	})

	t.Run("rejects nonexistent repository directory", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port: 8080,
			},
			Meta: MetaConfig{
				Host: "example.com",
			},
			Repo: RepoConfig{
				Dir:     "/nonexistent/path",
				Readmes: []string{"README.md"},
				Masters: []string{"main"},
			},
		}
		is.Err(t, cfg.validate(), "repo.dir")
	})

	t.Run("rejects empty readme list", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port: 8080,
			},
			Meta: MetaConfig{
				Host: "example.com",
			},
			Repo: RepoConfig{
				Dir:     tmpDir,
				Readmes: []string{},
				Masters: []string{"main"},
			},
		}
		is.Err(t, cfg.validate(), "repo.readmes")
	})

	t.Run("rejects empty master branches list", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port: 8080,
			},
			Meta: MetaConfig{
				Host: "example.com",
			},
			Repo: RepoConfig{
				Dir:     tmpDir,
				Readmes: []string{"README.md"},
				Masters: []string{},
			},
		}
		is.Err(t, cfg.validate(), "repo.masters")
	})

	t.Run("rejects invalid SSH port when SSH enabled", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port: 8080,
			},
			Meta: MetaConfig{
				Host: "example.com",
			},
			Repo: RepoConfig{
				Dir:     tmpDir,
				Readmes: []string{"README.md"},
				Masters: []string{"main"},
			},
			SSH: SSHConfig{
				Enable:  true,
				Port:    0,
				HostKey: hostKeyPath,
				Keys:    []string{"ssh-rsa AAAAB3..."},
			},
		}
		is.Err(t, cfg.validate(), "ssh.port")
	})

	t.Run("rejects SSH port same as HTTP port", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port: 8080,
			},
			Meta: MetaConfig{
				Host: "example.com",
			},
			Repo: RepoConfig{
				Dir:     tmpDir,
				Readmes: []string{"README.md"},
				Masters: []string{"main"},
			},
			SSH: SSHConfig{
				Enable:  true,
				Port:    8080,
				HostKey: hostKeyPath,
				Keys:    []string{"ssh-rsa AAAAB3..."},
			},
		}
		is.Err(t, cfg.validate(), "must differ")
	})

	t.Run("rejects nonexistent SSH host key file", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port: 8080,
			},
			Meta: MetaConfig{
				Host: "example.com",
			},
			Repo: RepoConfig{
				Dir:     tmpDir,
				Readmes: []string{"README.md"},
				Masters: []string{"main"},
			},
			SSH: SSHConfig{
				Enable:  true,
				Port:    2222,
				HostKey: "/nonexistent/key",
				Keys:    []string{"ssh-rsa AAAAB3..."},
			},
		}
		is.Err(t, cfg.validate(), "ssh.host_key")
	})

	t.Run("rejects empty SSH keys list when SSH enabled", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port: 8080,
			},
			Meta: MetaConfig{
				Host: "example.com",
			},
			Repo: RepoConfig{
				Dir:     tmpDir,
				Readmes: []string{"README.md"},
				Masters: []string{"main"},
			},
			SSH: SSHConfig{
				Enable:  true,
				Port:    2222,
				HostKey: hostKeyPath,
				Keys:    []string{},
			},
		}
		is.Err(t, cfg.validate(), "ssh.keys")
	})

	t.Run("rejects empty mirror interval", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port: 8080,
			},
			Meta: MetaConfig{
				Host: "example.com",
			},
			Repo: RepoConfig{
				Dir:     tmpDir,
				Readmes: []string{"README.md"},
				Masters: []string{"main"},
			},
			Mirror: MirrorConfig{
				Enable:   true,
				Interval: "",
			},
		}
		is.Err(t, cfg.validate(), "mirror.interval")
	})

	t.Run("rejects invalid mirror interval format", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port: 8080,
			},
			Meta: MetaConfig{
				Host: "example.com",
			},
			Repo: RepoConfig{
				Dir:     tmpDir,
				Readmes: []string{"README.md"},
				Masters: []string{"main"},
			},
			Mirror: MirrorConfig{
				Enable:   true,
				Interval: "1hour",
			},
		}
		is.Err(t, cfg.validate(), "invalid duration")
	})

	t.Run("collects and reports multiple validation errors", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port: 0,
			},
			Meta: MetaConfig{
				Host: "",
			},
			Repo: RepoConfig{
				Dir:     "/nonexistent",
				Readmes: []string{},
				Masters: []string{},
			},
		}
		err := cfg.validate()
		is.Err(t, err, "server.port")
		is.Err(t, err, "meta.host")
		is.Err(t, err, "repo.dir")
		is.Err(t, err, "repo.readmes")
		is.Err(t, err, "repo.masters")
	})

	t.Run("accepts multiple readme and master branch names", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port: 8080,
			},
			Meta: MetaConfig{
				Host: "example.com",
			},
			Repo: RepoConfig{
				Dir:     tmpDir,
				Readmes: []string{"README.md", "readme.txt", "README"},
				Masters: []string{"main", "master", "trunk"},
			},
		}
		is.Err(t, cfg.validate(), nil)
	})

	t.Run("ignores invalid SSH fields when SSH disabled", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port: 8080,
			},
			Meta: MetaConfig{
				Host: "example.com",
			},
			Repo: RepoConfig{
				Dir:     tmpDir,
				Readmes: []string{"README.md"},
				Masters: []string{"main"},
			},
			SSH: SSHConfig{
				Enable:  false,
				Port:    0,
				HostKey: "/nonexistent",
				Keys:    []string{},
			},
		}
		is.Err(t, cfg.validate(), nil)
	})

	t.Run("ignores invalid mirror fields when mirroring disabled", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port: 8080,
			},
			Meta: MetaConfig{
				Host: "example.com",
			},
			Repo: RepoConfig{
				Dir:     tmpDir,
				Readmes: []string{"README.md"},
				Masters: []string{"main"},
			},
			Mirror: MirrorConfig{
				Enable:   false,
				Interval: "invalid",
			},
		}
		is.Err(t, cfg.validate(), nil)
	})

	t.Run("accepts various time duration formats", func(t *testing.T) {
		durations := []string{"1h", "30m", "1h30m", "1h30m45s", "24h"}
		for _, duration := range durations {
			cfg := Config{
				Server: ServerConfig{
					Port: 8080,
				},
				Meta: MetaConfig{
					Host: "example.com",
				},
				Repo: RepoConfig{
					Dir:     tmpDir,
					Readmes: []string{"README.md"},
					Masters: []string{"main"},
				},
				Mirror: MirrorConfig{
					Enable:   true,
					Interval: duration,
				},
			}
			is.Err(t, cfg.validate(), nil)
		}
	})
}
