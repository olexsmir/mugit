package git

import (
	"strings"
	"testing"

	"olexsmir.xyz/x/is"
)

func TestRepo_Compare(t *testing.T) {
	t.Run("compares two refs", func(t *testing.T) {
		r := newTestRepo(t)
		base := r.commitFile("README.md", "base\n", "base commit")
		r.createBranch("develop", base)

		r.commitFile("master.txt", "master only\n", "master change")
		r.checkoutBranch("develop", false)
		r.commitFile("develop.txt", "develop only\n", "develop change")

		cmp, err := r.open().Compare("master", "develop")
		is.Err(t, err, nil)
		is.Equal(t, cmp.BaseRef, "master")
		is.Equal(t, cmp.HeadRef, "develop")
		is.Equal(t, cmp.Behind, 1)
		is.Equal(t, cmp.Ahead, 1)
		is.Equal(t, cmp.MergeBase, base.String())
		is.Equal(t, len(cmp.Commits), 1)
		is.Equal(t, cmp.Commits[0].Message, "develop change")
		is.Equal(t, cmp.Diff.Stat.FilesChanged, 1)
		is.Equal(t, cmp.Diff.Diff[0].Name.New, "develop.txt")
	})

	t.Run("returns empty range when refs are equal", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("README.md", "base\n", "base commit")

		cmp, err := r.open().Compare("master", "master")
		is.Err(t, err, nil)
		is.Equal(t, cmp.Behind, 0)
		is.Equal(t, cmp.Ahead, 0)
		is.Equal(t, len(cmp.Commits), 0)
		is.Equal(t, cmp.Diff.Stat.FilesChanged, 0)
	})

	t.Run("fails on invalid ref", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("README.md", "base\n", "base commit")

		_, err := r.open().Compare("master", "does-not-exist")
		is.Equal(t, strings.Contains(err.Error(), "resolving head ref"), true)
	})
}
