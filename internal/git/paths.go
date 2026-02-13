package git

import (
	"fmt"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
)

func ResolveName(name string) string {
	return strings.TrimSuffix(name, ".git") + ".git"
}

func ResolvePath(baseDir, repoName string) (string, error) {
	path, err := securejoin.SecureJoin(baseDir, repoName)
	if err != nil {
		return "", fmt.Errorf("failed to secure join paths: %w", err)
	}
	return path, err
}
