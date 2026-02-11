package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

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

// Load loads configuration with the following priority:
// 1. User provided fpath (if provided and exists)
// 2. /var/lib/mugit/config.yaml
// 3. $XDG_CONFIG_HOME/mugit/config.yaml or $HOME/.config/mugit/config.yaml
func Load(fpath string) (*Config, error) {
	// 4. /etc/mugit/config.yaml
	configPath, err := findConfigFile(fpath)
	if err != nil {
		return nil, err
	}

	configBytes, err := os.ReadFile(configPath)
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

func (c Config) validate() error {
	var errs []error

	// server
	if err := validatePort(c.Server.Port, "server.port"); err != nil {
		errs = append(errs, err)
	}

	// meta
	if c.Meta.Host == "" {
		errs = append(errs, errors.New("meta.host is required"))
	}

	// repo
	if err := validateDirExists(c.Repo.Dir, "repo.dir"); err != nil {
		errs = append(errs, err)
	}
	if len(c.Repo.Readmes) == 0 {
		errs = append(errs, errors.New("repo.readmes must have at least one value"))
	}
	if len(c.Repo.Masters) == 0 {
		errs = append(errs, errors.New("repo.masters must have at least one value"))
	}

	// ssh
	if c.SSH.Enable {
		if err := validatePort(c.SSH.Port, "ssh.port"); err != nil {
			errs = append(errs, err)
		}
		if c.SSH.Port == c.Server.Port {
			errs = append(errs, fmt.Errorf("ssh.port must differ from server.port (both are %d)", c.Server.Port))
		}
		if err := validateFileExists(c.SSH.HostKey, "ssh.host_key"); err != nil {
			errs = append(errs, err)
		}
		if len(c.SSH.Keys) == 0 {
			errs = append(errs, errors.New("ssh.keys must have at least one value when ssh is enabled"))
		}
	}

	// mirror
	if c.Mirror.Enable {
		if c.Mirror.Interval == "" {
			errs = append(errs, errors.New("mirror.interval is required when mirror is enabled"))
		} else if _, err := time.ParseDuration(c.Mirror.Interval); err != nil {
			errs = append(errs, fmt.Errorf("mirror.interval: invalid duration format: %w", err))
		}
	}

	return errors.Join(errs...)
}

func findConfigFile(userPath string) (string, error) {
	if userPath != "" {
		if _, err := os.Stat(userPath); err == nil {
			return userPath, nil
		}
	}

	path := "/var/lib/mugit/config.yaml"
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	if configDir, err := os.UserConfigDir(); err == nil {
		p := filepath.Join(configDir, "mugit", "config.yaml")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	path = "/etc/mugit/config.yaml"
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	return "", ErrConfigNotFound
}

func validatePort(port int, fieldName string) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%s must be between 1 and 65535, got %d", fieldName, port)
	}
	return nil
}

func validateDirExists(path string, fieldName string) error {
	if path == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s: directory does not exist: %s", fieldName, path)
		}
		return fmt.Errorf("%s: cannot access directory: %w", fieldName, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s: path exists but is not a directory: %s", fieldName, path)
	}
	return nil
}

func validateFileExists(path string, fieldName string) error {
	if path == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s: file does not exist: %s", fieldName, path)
		}
		return fmt.Errorf("%s: cannot access file: %w", fieldName, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s: path is a directory, not a file: %s", fieldName, path)
	}
	return nil
}
