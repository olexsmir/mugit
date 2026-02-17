package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

var (
	ErrConfigNotFound = errors.New("no config file found")
	ErrUnsetEnv       = errors.New("environment variable is not set")
	ErrFileNotFound   = errors.New("provided file path is invalid")
)

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
	Enable      bool          `yaml:"enable"`
	Interval    time.Duration `yaml:"interval"`
	GithubToken string        `yaml:"github_token"`
}

type CacheConfig struct {
	HomePage time.Duration `yaml:"home_page"`
	Readme   time.Duration `yaml:"readme"`
}

type Config struct {
	Server ServerConfig `yaml:"server"`
	Meta   MetaConfig   `yaml:"meta"`
	Repo   RepoConfig   `yaml:"repo"`
	SSH    SSHConfig    `yaml:"ssh"`
	Mirror MirrorConfig `yaml:"mirror"`
	Cache  CacheConfig  `yaml:"cache"`
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

	if perr := config.parseValues(); perr != nil {
		return nil, perr
	}

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
		c.Meta.Title = "my mugit"
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
	if c.Mirror.Interval == 0 {
		c.Mirror.Interval = 8 * time.Hour
	}

	// cache
	if c.Cache.HomePage == 0 {
		c.Cache.HomePage = 5 * time.Minute
	}

	if c.Cache.Readme == 0 {
		c.Cache.Readme = 1 * time.Minute
	}
}

func (c *Config) parseValues() error {
	if c.Mirror.Enable {
		ghToken, err := parseValue(c.Mirror.GithubToken)
		if err != nil {
			return err
		}
		c.Mirror.GithubToken = ghToken
	}
	return nil
}

func parseValue(value string) (string, error) {
	envPrefix := "$env:"
	filePrefix := "$file:"

	switch {
	case strings.HasPrefix(value, envPrefix):
		env := os.Getenv(os.ExpandEnv(value[len(envPrefix):]))
		if env == "" {
			return "", ErrUnsetEnv
		}
		return env, nil

	case strings.HasPrefix(value, filePrefix):
		// supports only absolute paths

		fpath := value[len(filePrefix):]
		if !isFileExists(fpath) {
			return "", ErrFileNotFound
		}

		data, err := os.ReadFile(fpath)
		if err != nil {
			return "", err
		}

		return strings.TrimSpace(string(data)), nil

	default:
		return value, nil
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
