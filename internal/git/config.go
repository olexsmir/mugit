package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	gitconfig "github.com/go-git/go-git/v5/config"
)

func (g *Repo) IsPrivate() (bool, error) {
	v, err := g.readOption("private")
	if err != nil {
		return false, err
	}
	return v == "true", nil
}

const originRemote = "origin"

func (g *Repo) IsMirror() (bool, error) {
	r, err := g.r.Remote(originRemote)
	if err != nil {
		return false, fmt.Errorf("failed to get remote: %w", err)
	}
	return r.Config().Mirror, nil
}

func (g *Repo) SetMirrorRemote(url string) error {
	_, err := g.r.CreateRemote(&gitconfig.RemoteConfig{
		Name:   originRemote,
		URLs:   []string{url},
		Mirror: true,
		Fetch: []gitconfig.RefSpec{
			"+refs/*:refs/*",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create origin remote: %w", err)
	}
	return nil
}

func (g *Repo) RemoteURL() (string, error) {
	r, err := g.r.Remote(originRemote)
	if err != nil {
		return "", fmt.Errorf("failed to get remote: %w", err)
	}
	return r.Config().URLs[0], nil
}

const defaultDescription = "Unnamed repository; edit this file 'description' to name the repository"

func (g *Repo) Description() (string, error) {
	path := filepath.Join(g.path, "description")
	if _, err := os.Stat(path); err != nil {
		return "", nil
	}

	d, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read description file: %w", err)
	}

	desc := string(d)
	if strings.Contains(desc, defaultDescription) {
		return "", nil
	}
	return desc, nil
}

func (g *Repo) SetDescription(desc string) error {
	path := filepath.Join(g.path, "description")
	return os.WriteFile(path, []byte(desc), 0o644)
}

func (g *Repo) LastSync() (time.Time, error) {
	raw, err := g.readOption("last-sync")
	if err != nil {
		return time.Time{}, err
	}

	if raw == "" {
		return time.Time{}, fmt.Errorf("last-sync not set")
	}

	out, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse time: %w", err)
	}

	return out, nil
}

func (g *Repo) SetLastSync(lastSync time.Time) error {
	return g.setOption("last-sync", lastSync.Format(time.RFC3339))
}

func (g *Repo) readOption(key string) (string, error) {
	c, err := g.r.Config()
	if err != nil {
		return "", fmt.Errorf("failed to read config: %w", err)
	}
	return c.Raw.Section("mugit").Options.Get(key), nil
}

func (g *Repo) setOption(key, value string) error {
	c, err := g.r.Config()
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}
	c.Raw.Section("mugit").SetOption(key, value)
	return g.r.SetConfig(c)
}
