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

// topLevelEntry returns the top-level entry name under base for a given path.
// e.g. base="lua", path="lua/plugins/foo.lua" -> "plugins"
// e.g. base="",    path="README.md"           -> "README.md"
// returns "" if path is not under base.
func topLevelEntry(fullPath, base string) string {
	if base != "" && base != "." {
		if !strings.HasPrefix(fullPath, base+"/") {
			return ""
		}
		fullPath = fullPath[len(base)+1:]
	}
	if before, _, ok := strings.Cut(fullPath, "/"); ok {
		return before
	}
	return fullPath
}
