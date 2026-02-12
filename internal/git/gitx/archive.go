package gitx

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
)

// ArchiveTar generates a tarball of a git ref.
func ArchiveTar(ctx context.Context, repoDir, ref string, out io.Writer) error {
	if !isValidRef(ref) {
		return fmt.Errorf("invalid ref: %s", ref)
	}

	cmd := exec.CommandContext(ctx, "git", "archive", "--format=tar.gz", ref)
	cmd.Dir = repoDir
	cmd.Env = gitEnv
	cmd.Stdout = out
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git archive %s: %w", ref, err)
	}

	return nil
}

func isValidRef(ref string) bool {
	if ref == "" || strings.Contains(ref, "..") {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9._/-]+$`, ref)
	return matched
}
