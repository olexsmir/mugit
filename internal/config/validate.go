package config

import (
	"errors"
	"fmt"
	"time"
)

func (c Config) validate() error {
	var errs []error

	if c.Meta.Host == "" {
		// TODO: actually it should be a warning, host only used for go-import tag
		errs = append(errs, errors.New("meta.host is required"))
	}

	if !isDirExists(c.Repo.Dir) {
		errs = append(errs, fmt.Errorf("repo.dir seems to be an invalid path"))
	}

	if err := checkPort(c.Server.Port); err != nil {
		errs = append(errs, fmt.Errorf("server.port %w", err))
	}

	if c.SSH.Enable {
		if err := checkPort(c.SSH.Port); err != nil {
			errs = append(errs, fmt.Errorf("ssh.port %w", err))
		}

		if c.SSH.Port == c.Server.Port {
			errs = append(errs, fmt.Errorf("ssh.port must differ from server.port (both are %d)", c.Server.Port))
		}

		if !isFileExists(c.SSH.HostKey) {
			errs = append(errs, fmt.Errorf("ssh.host_key seems to be an invalid path"))
		}
	}

	if c.Mirror.Enable {
		if _, err := time.ParseDuration(c.Mirror.Interval); err != nil {
			errs = append(errs, fmt.Errorf("mirror.interval: invalid duration format: %w", err))
		}
	}

	return errors.Join(errs...)
}

func checkPort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("must be between 1 and 65535, got %d", port)
	}
	return nil
}
