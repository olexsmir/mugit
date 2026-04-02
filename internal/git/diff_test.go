package git

import (
	"testing"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"olexsmir.xyz/x/is"
)

func TestRepo_Diff(t *testing.T) {
	t.Run("single file addition", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("README.md", "# Test", "Initial commit")
		r.commitFile("hello.txt", "hello world\n", "Add hello file")

		diff, err := r.open().Diff()
		is.Err(t, err, nil)
		is.Equal(t, diff.Stat.FilesChanged, 1)
		is.Equal(t, diff.Stat.Insertions, 1)
		is.Equal(t, diff.Stat.Deletions, 0)
		is.Equal(t, len(diff.Diff), 1)
		is.Equal(t, diff.Diff[0].Name.New, "hello.txt")
		is.Equal(t, diff.Diff[0].IsNew, true)
	})

	t.Run("file modification", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("README.md", "# Original\n", "Initial commit")
		r.commitFile("README.md", "# Modified\n\nNew content here.\n", "Update README")

		diff, err := r.open().Diff()
		is.Err(t, err, nil)
		is.Equal(t, diff.Stat.FilesChanged, 1)
		is.Equal(t, diff.Diff[0].Name.New, "README.md")
		is.Equal(t, diff.Diff[0].IsNew, false)
		is.Equal(t, diff.Diff[0].IsDelete, false)
		if diff.Stat.Insertions == 0 {
			t.Error("expected insertions > 0")
		}
		if diff.Stat.Deletions == 0 {
			t.Error("expected deletions > 0")
		}
	})

	t.Run("file deletion", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("todelete.txt", "temp content\n", "Add temp file")
		r.deleteFile("todelete.txt", "Delete temp file")

		diff, err := r.open().Diff()
		is.Err(t, err, nil)
		is.Equal(t, diff.Stat.FilesChanged, 1)
		is.Equal(t, diff.Stat.Deletions, 1)
		is.Equal(t, diff.Diff[0].IsDelete, true)
	})

	t.Run("multiple files changed", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("file1.txt", "content 1\n", "Add file1")
		r.commitFile("file2.txt", "content 2\n", "Add file2")
		r.commitFile("file3.txt", "content 3\n", "Add file3")

		diff, err := r.open().Diff()
		is.Err(t, err, nil)
		is.Equal(t, diff.Stat.FilesChanged, 1)
		is.Equal(t, diff.Stat.Insertions, 1)
	})

	t.Run("has parent hashes", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("first.txt", "first\n", "First commit")
		r.commitFile("second.txt", "second file\n", "Add second file")

		diff, err := r.open().Diff()
		is.Err(t, err, nil)
		is.Equal(t, len(diff.Parents), 1)
		if len(diff.Parents[0]) == 0 {
			t.Error("expected parent hash to be non-empty")
		}
	})

	t.Run("initial commit has no parents", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("initial.txt", "initial\n", "Initial commit")

		commits, err := r.open().Commits("")
		is.Err(t, err, nil)
		if len(commits) == 0 {
			t.Fatal("expected at least one commit")
		}

		initial := r.open(commits[len(commits)-1].Hash)
		diff, err := initial.Diff()
		is.Equal(t, len(diff.Parents), 0)
		is.Err(t, err, nil)
	})

	t.Run("text fragments have line info", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("README.md", "original\n", "Initial commit")
		r.commitFile("README.md", "line 1\nline 2\nline 3\n", "Multi-line change")

		diff, err := r.open().Diff()
		is.Err(t, err, nil)
		if len(diff.Diff) == 0 {
			t.Fatal("expected at least one diff")
		}
		if len(diff.Diff[0].TextFragments) == 0 {
			t.Fatal("expected at least one text fragment")
		}

		frag := diff.Diff[0].TextFragments[0]
		if len(frag.Lines) == 0 {
			t.Fatal("expected at least one line")
		}

		// Check that lines have operations
		hasAdd := false
		for _, line := range frag.Lines {
			if line.Op == gitdiff.OpAdd {
				hasAdd = true
			}
		}
		if !hasAdd {
			t.Error("expected at least one added line")
		}
	})

	t.Run("commit info is populated", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("info.txt", "test\n", "Test commit message")

		diff, err := r.open().Diff()
		is.Err(t, err, nil)
		is.Equal(t, diff.Commit.Message, "Test commit message")
		is.Equal(t, diff.Commit.AuthorName, "Test User")
		is.Equal(t, diff.Commit.AuthorEmail, "test@test.local")
		if len(diff.Commit.Hash) == 0 {
			t.Error("expected commit hash to be non-empty")
		}
	})
}

func TestTextFragment(t *testing.T) {
	frag := TextFragment{
		Header:      "@@ -1,3 +1,4 @@",
		OldPosition: 1,
		NewPosition: 1,
		Lines: []gitdiff.Line{
			{Op: gitdiff.OpContext, Line: "context line"},
			{Op: gitdiff.OpAdd, Line: "added line"},
		},
	}
	is.Equal(t, frag.Header, "@@ -1,3 +1,4 @@")
	is.Equal(t, frag.OldPosition, int64(1))
	is.Equal(t, frag.NewPosition, int64(1))
	is.Equal(t, len(frag.Lines), 2)
}
