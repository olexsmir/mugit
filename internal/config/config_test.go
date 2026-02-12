package config

import (
	"os"
	"path/filepath"
	"testing"

	"olexsmir.xyz/x/is"
)

func TestConfig_parseValue(t *testing.T) {
	def := "qwerty123"

	t.Run("string", func(t *testing.T) {
		r, err := parseValue(def)
		is.Err(t, err, nil)
		is.Equal(t, r, def)
	})

	t.Run("env var", func(t *testing.T) {
		t.Setenv("secret_value", "123")
		r, err := parseValue("$env:secret_value")
		is.Err(t, err, nil)
		is.Equal(t, r, "123")
	})

	t.Run("unset env var", func(t *testing.T) {
		_, err := parseValue("$env:secret_password")
		is.Err(t, err, ErrUnsetEnv)
	})

	t.Run("file", func(t *testing.T) {
		fpath, _ := filepath.Abs("./testdata/file_value")
		r, err := parseValue("$file:" + fpath)
		is.Err(t, err, nil)
		is.Equal(t, r, def)
	})

	t.Run("non existing file", func(t *testing.T) {
		_, err := parseValue("$file:/not/exists")
		is.Err(t, err, ErrFileNotFound)
	})

	t.Run("file, not set path", func(t *testing.T) {
		_, err := parseValue("$file:")
		is.Err(t, err, ErrFileNotFound)
	})
}

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
