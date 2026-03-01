package gitx

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// ArchiveTar generates a tarball of a git ref.
func ArchiveTar(ctx context.Context, repoDir, ref string, out io.Writer) error {
	if !isValidRef(ref) {
		return fmt.Errorf("invalid ref: %s", ref)
	}

	if err := gitCmd(ctx, cmdOpts{
		Cmd:     []string{"archive", "--format=tar.gz", ref},
		RepoDir: repoDir,
		Stdout:  out,
	}); err != nil {
		return fmt.Errorf("git archive %s: %w", ref, err)
	}

	return nil
}

func UploadArchive(ctx context.Context, repoDir string, in io.Reader, out io.Writer) error {
	if err := gitCmd(ctx, cmdOpts{
		RepoDir: repoDir,
		Cmd:     []string{"upload-archive"},
		Stdin:   in,
		Stdout:  out,
		Stderr:  out,
	}); err != nil {
		return fmt.Errorf("git-upload-archive: %w", err)
	}
	return nil
}

var isValidRefRe = regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)

func isValidRef(ref string) bool {
	if ref == "" || strings.Contains(ref, "..") {
		return false
	}
	return isValidRefRe.MatchString(ref)
}
