package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

var ErrConfigNotFound = errors.New("no config file found")

type Config struct {
	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"server"`
	Meta struct {
		Title       string `yaml:"title"`
		Description string `yaml:"description"`
		Host        string `yaml:"host"`
	} `yaml:"meta"`
	Repo struct {
		Dir     string   `yaml:"dir"`
		Readmes []string `yaml:"readmes"`
		Masters []string `yaml:"masters"`
	} `yaml:"repo"`
	SSH struct {
		Enable  bool     `yaml:"enable"`
		Port    int      `yaml:"port"`
		HostKey string   `yaml:"host_key"`
		Keys    []string `yaml:"keys"`
	} `yaml:"ssh"`
	Mirror struct {
		Enable      bool   `yaml:"enable"`
		Interval    string `yaml:"interval"`
		GithubToken string `yaml:"github_token"`
	} `yaml:"mirror"`
}

// Load loads configuration with the following priority:
// 1. User provided fpath (if provided and exists)
// 2. $XDG_CONFIG_HOME/mugit/config.yaml
// 3. $HOME/.config/mugit/config.yaml (fallback if XDG_CONFIG_HOME not set)
// 4. /etc/mugit/config.yaml
func Load(fpath string) (*Config, error) {
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

	if verr := config.validate(); verr != nil {
		return nil, verr
	}

	return &config, nil
}

func (c Config) validate() error {
	// var errs []error
	// return errors.Join(errs...)
	return nil
}

func findConfigFile(userPath string) (string, error) {
	if userPath != "" {
		if _, err := os.Stat(userPath); err == nil {
			return userPath, nil
		}
	}

	paths := []string{}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		paths = append(paths, filepath.Join(xdg, "mugit", "config.yaml"))
	} else if home := os.Getenv("HOME"); home != "" {
		paths = append(paths, filepath.Join(home, ".config", "mugit", "config.yaml"))
	}

	paths = append(paths, "/etc/mugit/config.yaml")
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", ErrConfigNotFound
}
