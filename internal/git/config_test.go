package git

import (
	"os"
	"path/filepath"
	"testing"

	"olexsmir.xyz/x/is"
)

func TestRepo_IsPrivate(t *testing.T) {
	t.Run("default is not private", func(t *testing.T) {
		tr := newTestRepo(t)
		private, err := tr.open().IsPrivate()
		is.Equal(t, private, false)
		is.Err(t, err, nil)
	})

	t.Run("set to private", func(t *testing.T) {
		r := newTestRepo(t).open()
		is.Err(t, r.SetPrivate(true), nil)

		private, err := r.IsPrivate()
		is.Err(t, err, nil)
		is.Equal(t, private, true)
	})

	t.Run("can be set back to public", func(t *testing.T) {
		r := newTestRepo(t).open()

		is.Err(t, r.SetPrivate(true), nil)
		is.Err(t, r.SetPrivate(false), nil)

		private, err := r.IsPrivate()
		is.Equal(t, private, false)
		is.Err(t, err, nil)
	})
}

func TestRepo_Description(t *testing.T) {
	t.Run("default description is empty description", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("dummy.txt", "dummy", "Initial commit")

		descPath := filepath.Join(r.path, ".git", "description")
		_ = os.WriteFile(descPath, []byte(defaultDescription), 0o644)

		desc, err := r.open().Description()
		is.Err(t, err, nil)
		is.Equal(t, desc, "")
	})

	t.Run("empty description returns empty string", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("dummy.txt", "dummy", "Initial commit")

		desc, err := r.open().Description()
		is.Err(t, err, nil)
		is.Equal(t, desc, "")
	})

	t.Run("default git description is treated as empty", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("dummy.txt", "dummy", "Initial commit")

		// Write the default git description to the .git directory
		descPath := filepath.Join(r.path, ".git", "description")
		err := os.WriteFile(descPath, []byte(defaultDescription), 0o644)
		is.Err(t, err, nil)

		desc, err := r.open().Description()
		is.Err(t, err, nil)
		is.Equal(t, desc, "")
	})

	t.Run("set and get description", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("dummy.txt", "dummy", "Initial commit")

		repo := r.open()
		err := repo.SetDescription("My awesome project")
		is.Err(t, err, nil)

		desc, err := repo.Description()
		is.Err(t, err, nil)
		is.Equal(t, desc, "My awesome project")
	})

	t.Run("description with newlines", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("dummy.txt", "dummy", "Initial commit")

		repo := r.open()
		multiLine := "My project\n\nWith multiple lines\nof description."
		err := repo.SetDescription(multiLine)
		is.Err(t, err, nil)

		desc, err := repo.Description()
		is.Err(t, err, nil)
		is.Equal(t, desc, multiLine)
	})
}

func TestRepo_Mirror(t *testing.T) {
	t.Run("repo without origin remote can't be a mirror", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("dummy.txt", "dummy", "Initial commit")

		_, err := r.open().IsMirror()
		is.Err(t, err, "failed to get remote: ")
	})

	t.Run("set mirror remote", func(t *testing.T) {
		r := newTestRepo(t).open()

		expectedURL := "https://github.com/example/repo.git"
		err := r.SetMirrorRemote(expectedURL)
		is.Err(t, err, nil)

		isMirror, err := r.IsMirror()
		is.Equal(t, isMirror, true)
		is.Err(t, err, nil)

		url, err := r.RemoteURL()
		is.Err(t, err, nil)
		is.Equal(t, url, expectedURL)
	})
}
