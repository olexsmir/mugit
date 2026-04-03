package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"olexsmir.xyz/x/is"
)

func TestRepo_Name(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "name", path: "/repos/myrepo", want: "myrepo"},
		{name: "with .git", path: "/repos/myrepo.git", want: "myrepo"},
		{name: "nested path", path: "/home/user/code/project", want: "project"},
		{name: "nested with .git", path: "/home/user/repos/awesome-project.git", want: "awesome-project"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Repo{path: tt.path}
			is.Equal(t, r.Name(), tt.want)
		})
	}
}

func TestRepo_IsEmpty(t *testing.T) {
	t.Run("empty repo", func(t *testing.T) {
		r := &Repo{h: plumbing.ZeroHash}
		is.Equal(t, r.IsEmpty(), true)
	})

	t.Run("non-empty repo", func(t *testing.T) {
		r := &Repo{h: plumbing.NewHash("abc123def456789abc123def456789abc123def4")}
		is.Equal(t, r.IsEmpty(), false)
	})
}

func TestInit(t *testing.T) {
	t.Run("creates bare repo", func(t *testing.T) {
		dir := t.TempDir()
		repoPath := filepath.Join(dir, "test.git")

		err := Init(repoPath)
		is.Err(t, err, nil)

		_, err = os.Stat(filepath.Join(repoPath, "HEAD"))
		is.Err(t, err, nil)
	})

	t.Run("fails on existing repo", func(t *testing.T) {
		dir := t.TempDir()
		repoPath := filepath.Join(dir, "test.git")

		err := Init(repoPath)
		is.Err(t, err, nil)

		is.Err(t, Init(repoPath), "repository already exists")
	})
}

func TestOpen(t *testing.T) {
	t.Run("opens repo at HEAD", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("README.md", "Test", "Initial commit")

		repo, err := Open(r.path, "")
		is.Equal(t, repo.IsEmpty(), false)
		is.Err(t, err, nil)
	})

	t.Run("opens repo at specific ref", func(t *testing.T) {
		r := newTestRepo(t)
		firstHash := r.commitFile("file1.txt", "first", "first commit")
		r.commitFile("file2.txt", "second", "second commit")

		repo := r.open(firstHash.String())
		commit, err := repo.LastCommit()
		is.Equal(t, commit.Message, "first commit")
		is.Err(t, err, nil)
	})

	t.Run("fails on invalid path", func(t *testing.T) {
		_, err := Open("/nonexistent/path", "")
		is.Err(t, err, "does not exist")
	})

	t.Run("fails on invalid ref", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("README.md", "# Test", "Initial commit")

		_, err := Open(r.path, "nonexistent-ref")
		is.Err(t, err, "resolving rev ")
	})
}

func TestOpenPublic(t *testing.T) {
	t.Run("opens public repo", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("README.md", "# Test", "Initial commit")

		repo, err := OpenPublic(r.path, "")
		is.Equal(t, repo.IsEmpty(), false)
		is.Err(t, err, nil)
	})

	t.Run("returns ErrPrivate for private repo", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("README.md", "# Test", "Initial commit")

		err := r.open().SetPrivate(true)
		is.Err(t, err, nil)

		_, err = OpenPublic(r.path, "")
		is.Err(t, err, ErrPrivate)
	})
}

func TestRepo_Commits(t *testing.T) {
	t.Run("returns commits in reverse chronological order", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("README.md", "# Test", "Initial commit")
		r.commitFile("a.txt", "a", "Add a")
		r.commitFile("b.txt", "b", "Add b")
		r.commitFile("c.txt", "c", "Add c")

		commits, err := r.open().Commits("")
		is.Err(t, err, nil)

		is.Equal(t, len(commits), 4)
		is.Equal(t, commits[0].Message, "Add c")
		is.Equal(t, commits[1].Message, "Add b")
		is.Equal(t, commits[2].Message, "Add a")
		is.Equal(t, commits[3].Message, "Initial commit")
	})

	t.Run("pagination with after cursor", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("README.md", "# Test", "Initial commit")
		r.commitFile("a.txt", "a", "Add a")
		r.commitFile("b.txt", "b", "Add b")

		// get all commits first
		all, err := r.open().Commits("")
		is.Equal(t, len(all), 3)
		is.Err(t, err, nil)

		// get commits after the first one
		after, err := r.open().Commits(all[0].HashShort)
		is.Err(t, err, nil)
		is.Equal(t, len(after), 2)
		is.Equal(t, after[0].Message, "Add a")
	})

	t.Run("empty repo returns empty slice", func(t *testing.T) {
		r := newTestRepo(t)
		commits, err := r.open().Commits("")
		is.Equal(t, len(commits), 0)
		is.Err(t, err, nil)
	})
}

func TestRepo_LastCommit(t *testing.T) {
	t.Run("returns HEAD commit", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("readme", "test", "init")
		r.commitFile("latest.txt", "latest", "latest commit")

		commit, err := r.open().LastCommit()
		is.Equal(t, commit.Message, "latest commit")
		is.Err(t, err, nil)
	})

	t.Run("empty repo returns empty commit", func(t *testing.T) {
		commit, err := newTestRepo(t).open().LastCommit()
		is.Err(t, err, nil)
		is.Equal(t, commit.Message, "")
	})
}

func TestRepo_Branches(t *testing.T) {
	r := newTestRepo(t)
	hash := r.commitFile("file.txt", "content", "A commit")
	r.createBranch("feature", hash)
	r.createBranch("develop", hash)

	branches, err := r.open().Branches()
	is.Err(t, err, nil)

	names := make(map[string]bool, len(branches))
	for _, b := range branches {
		names[b.Name] = true
	}

	is.Equal(t, names["master"], true) // got on init
	is.Equal(t, names["feature"], true)
	is.Equal(t, names["develop"], true)
}

func TestRepo_IsGoMod(t *testing.T) {
	t.Run("without go.mod", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("readme", "test", "init")
		is.Equal(t, r.open().IsGoMod(), false)
	})

	t.Run("with go.mod", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("go.mod", "module example.com/test\n\ngo 1.21\n", "Add go.mod")
		is.Equal(t, r.open().IsGoMod(), true)
	})
}

func TestRepo_DefaultBranch(t *testing.T) {
	t.Run("works", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("readme", "test", "init")

		branch, err := r.open().DefaultBranch()
		is.Equal(t, branch, "master")
		is.Err(t, err, nil)
	})

	t.Run("multiple branches", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("readme", "test", "init")
		m := r.commitFile("main", "test", "init2")
		r.createBranch("develop", m)

		branch, err := r.open().DefaultBranch()
		is.Equal(t, branch, "master")
		is.Err(t, err, nil)
	})
}

func TestRepo_SetDefaultBranch(t *testing.T) {
	r := newTestRepo(t)
	r.commitFile("readme", "test", "init")

	rr := r.open()

	branch, err := rr.DefaultBranch()
	is.Equal(t, branch, "master")
	is.Err(t, err, nil)

	h := r.commitFile("thing", "hello worldie", "new feature")
	r.createBranch("develop", h)
	rr.SetDefaultBranch("develop")

	branch, err = rr.DefaultBranch()
	is.Equal(t, branch, "develop")
	is.Err(t, err, nil)
}
