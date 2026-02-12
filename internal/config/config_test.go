package config

import (
	"os"
	"path/filepath"
	"testing"

	"olexsmir.xyz/x/is"
)

func TestPathOrDefaultWithCandidates(t *testing.T) {
	first := candidateFile(t, "first.yaml")
	second := candidateFile(t, "second.yaml")
	third := candidateFile(t, "third.yaml")

	t.Run("returns user path when exists", func(t *testing.T) {
		userPath := candidateFile(t, "user.yaml")
		candidates := []string{first, second, third}
		got := pathOrDefaultWithCandidates(userPath, candidates)
		if got != userPath {
			t.Errorf("got %q, want %q", got, userPath)
		}
	})

	t.Run("returns first existing candidate", func(t *testing.T) {
		candidates := []string{first, second, third}
		got := pathOrDefaultWithCandidates("", candidates)
		is.Equal(t, got, first)
	})

	t.Run("returns empty when nothing exists", func(t *testing.T) {
		candidates := []string{}
		got := pathOrDefaultWithCandidates("", candidates)
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

func candidateFile(t *testing.T, name string) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), name)
	os.WriteFile(out, []byte("test"), 0o644)
	return out
}
