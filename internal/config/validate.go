package config

import (
	"errors"
	"fmt"
	"strings"
)

func (c Config) validate() error {
	var errs []error

	if c.Meta.Host == "" {
		errs = append(errs, errors.New("meta.host is required"))
	}

	if strings.HasPrefix(c.Meta.Host, "http") {
		errs = append(errs, errors.New("meta.host shouldn't include protocol"))
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

	return errors.Join(errs...)
}

func checkPort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("must be between 1 and 65535, got %d", port)
	}
	return nil
}
