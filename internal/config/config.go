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
		Title        string `yaml:"title"`
		Description  string `yaml:"description"`
		Host         string `yaml:"host"`
		ChromaTheme  string `yaml:"chroma_theme"`
		TemplatesDir string `yaml:"templates_dir"`
	} `yaml:"meta"`
	Repo struct {
		Dir     string   `yaml:"dir"`
		Readmes []string `yaml:"readmes"`
		Masters []string `yaml:"masters"`
		Private []string `yaml:"private"`
	} `yaml:"repo"`
	SSH struct {
		Keys []string `yaml:"keys"`
	} `yaml:"ssh"`
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

	if config.Meta.TemplatesDir, err = filepath.Abs(config.Meta.TemplatesDir); err != nil {
		return nil, err
	}

	fmt.Println(config.Meta.TemplatesDir)

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
