package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"olexsmir.xyz/x/is"
)

type testRepo struct {
	tb   testing.TB
	r    *git.Repository
	path string
}

func newTestRepo(tb testing.TB) *testRepo {
	tb.Helper()
	path := tb.TempDir()

	// creates non-bare repo, bare repos require to keep track of HEAD manually
	r, err := git.PlainInit(path, false)
	is.Err(tb, err, nil)

	cfg, err := r.Config()
	is.Err(tb, err, nil)

	cfg.User.Name = "Test User"
	cfg.User.Email = "test@test.local"
	is.Err(tb, r.SetConfig(cfg), nil)

	return &testRepo{tb: tb, path: path, r: r}
}

func (t *testRepo) commitFileAt(name, content, msg string, when time.Time) plumbing.Hash {
	t.tb.Helper()

	filePath := filepath.Join(t.path, name)
	is.Err(t.tb, os.MkdirAll(filepath.Dir(filePath), 0o755), nil)
	is.Err(t.tb, os.WriteFile(filePath, []byte(content), 0o644), nil)

	wt, err := t.r.Worktree()
	is.Err(t.tb, err, nil)

	_, err = wt.Add(name)
	is.Err(t.tb, err, nil)

	hash, err := wt.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@test.local",
			When:  when,
		},
	})
	is.Err(t.tb, err, nil)
	return hash
}

func (t *testRepo) commitFile(name, content, msg string) plumbing.Hash {
	t.tb.Helper()
	return t.commitFileAt(name, content, msg, time.Now())
}

func (t *testRepo) deleteFile(name, msg string) plumbing.Hash {
	t.tb.Helper()

	wt, err := t.r.Worktree()
	is.Err(t.tb, err, nil)

	_, err = wt.Remove(name)
	is.Err(t.tb, err, nil)

	hash, err := wt.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@test.local",
			When:  time.Now(),
		},
	})
	is.Err(t.tb, err, nil)
	return hash
}

func (t *testRepo) createBranch(name string, hash plumbing.Hash) {
	t.tb.Helper()
	ref := plumbing.NewHashReference(plumbing.NewBranchReferenceName(name), hash)
	is.Err(t.tb, t.r.Storer.SetReference(ref), nil)
}

func (t *testRepo) createTag(name string, hash plumbing.Hash) {
	t.tb.Helper()
	ref := plumbing.NewHashReference(plumbing.NewTagReferenceName(name), hash)
	is.Err(t.tb, t.r.Storer.SetReference(ref), nil)
}

func (t *testRepo) createAnnotatedTag(name, msg string, hash plumbing.Hash, when time.Time) {
	t.tb.Helper()
	_, err := t.r.CreateTag(name, hash, &git.CreateTagOptions{
		Tagger: &object.Signature{
			Name:  "Test User",
			Email: "test@test.local",
			When:  when,
		},
		Message: msg,
	})
	is.Err(t.tb, err, nil)
}

func (t *testRepo) open(ref ...string) *Repo {
	t.tb.Helper()
	re := ""
	if len(ref) == 1 {
		re = ref[0]
	}
	r, err := Open(t.path, re)
	is.Err(t.tb, err, nil)
	return r
}
