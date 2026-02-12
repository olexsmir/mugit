package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

var ErrConfigNotFound = errors.New("no config file found")

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type MetaConfig struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Host        string `yaml:"host"`
}

type RepoConfig struct {
	Dir     string   `yaml:"dir"`
	Readmes []string `yaml:"readmes"`
	Masters []string `yaml:"masters"`
}

type SSHConfig struct {
	Enable  bool     `yaml:"enable"`
	Port    int      `yaml:"port"`
	HostKey string   `yaml:"host_key"`
	Keys    []string `yaml:"keys"`
}

type MirrorConfig struct {
	Enable      bool   `yaml:"enable"`
	Interval    string `yaml:"interval"`
	GithubToken string `yaml:"github_token"`
}

type Config struct {
	Server ServerConfig `yaml:"server"`
	Meta   MetaConfig   `yaml:"meta"`
	Repo   RepoConfig   `yaml:"repo"`
	SSH    SSHConfig    `yaml:"ssh"`
	Mirror MirrorConfig `yaml:"mirror"`
}

func Load(fpath string) (*Config, error) {
	configBytes, err := os.ReadFile(fpath)
	if err != nil {
		return nil, err
	}

	var config Config
	if cerr := yaml.Unmarshal(configBytes, &config); cerr != nil {
		return nil, fmt.Errorf("parsing config: %w", cerr)
	}

	if config.Repo.Dir, err = filepath.Abs(config.Repo.Dir); err != nil {
		return nil, err
	}

	config.ensureDefaults()

	if verr := config.validate(); verr != nil {
		return nil, verr
	}

	return &config, nil
}

// PathOrDefault uses userPath, if it's "", or invalid path, will default to one of those(in priority order)
// 1. ./config.yaml
// 2. /etc/mugit.yaml
// 3. /var/lib/mugit/config.yaml
func PathOrDefault(userPath string) string {
	return pathOrDefaultWithCandidates(userPath, []string{
		"./config.yaml",
		"/etc/mugit.yaml",
		"/var/lib/mugit/config.yaml",
	})
}

func pathOrDefaultWithCandidates(path string, candidates []string) string {
	if isFileExists(path) {
		return path
	}

	for _, fpath := range candidates {
		if isFileExists(fpath) {
			return fpath
		}
	}

	return ""
}

func (c *Config) ensureDefaults() {
	// ports
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}

	if c.SSH.Port == 0 {
		c.SSH.Port = 2222
	}

	// meta
	if c.Meta.Title == "" {
		c.Meta.Title = "my cgit"
	}

	// repos
	if len(c.Repo.Masters) == 0 {
		c.Repo.Masters = []string{"master", "main"}
	}

	if len(c.Repo.Readmes) == 0 {
		c.Repo.Readmes = []string{
			"README.md", "readme.md",
			"README.html", "readme.html",
			"README.txt", "readme.txt",
			"readme",
		}
	}

	// mirroring
	if c.Mirror.Interval == "" {
		c.Mirror.Interval = "8h"
	}
}

func isFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isDirExists(path string) bool {
	i, err := os.Stat(path)
	if err != nil {
		return false
	}
	return i.IsDir()
}
