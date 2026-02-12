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
