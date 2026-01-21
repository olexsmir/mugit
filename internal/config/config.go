package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

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

func Load(fpath string) (*Config, error) {
	configBytes, err := os.ReadFile(fpath)
	if err != nil {
		return nil, err
	}

	var config *Config
	if cerr := yaml.Unmarshal(configBytes, &config); cerr != nil {
		return nil, fmt.Errorf("parsing config: %w", cerr)
	}

	if config.Repo.Dir, err = filepath.Abs(config.Repo.Dir); err != nil {
		return nil, err
	}

	if verr := config.validate(); verr != nil {
		return nil, verr
	}

	return config, nil
}

func (c Config) validate() error {
	// var errs []error
	// return errors.Join(errs...)
	return nil
}
