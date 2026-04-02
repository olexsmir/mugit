package git

import (
	"testing"
	"time"

	"olexsmir.xyz/x/is"
)

func TestRepo_Tags(t *testing.T) {
	t.Run("empty repo has no tags", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("dummy.txt", "dummy", "Initial commit")

		tags, err := r.open().Tags()
		is.Equal(t, len(tags), 0)
		is.Err(t, err, nil)
	})

	t.Run("lightweight tag", func(t *testing.T) {
		r := newTestRepo(t)
		hash := r.commitFile("file.txt", "content", "A commit")
		r.createTag("v1.0.0", hash)

		tags, err := r.open().Tags()
		is.Err(t, err, nil)
		is.Equal(t, len(tags), 1)
		is.Equal(t, tags[0].Name(), "v1.0.0")
		is.Equal(t, tags[0].Message(), "")
	})

	t.Run("annotated tag", func(t *testing.T) {
		tr := newTestRepo(t)
		hash := tr.commitFile("file.txt", "content", "A commit")
		tagTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
		tr.createAnnotatedTag("v2.0.0", "Release version 2.0.0\n\nThis is a major release", hash, tagTime)

		tags, err := tr.open().Tags()
		is.Err(t, err, nil)
		is.Equal(t, len(tags), 1)
		is.Equal(t, tags[0].Name(), "v2.0.0")
		is.Equal(t, tags[0].Message(), "Release version 2.0.0\n\nThis is a major release\n")
		is.Equal(t, tags[0].When(), tagTime)
	})

	t.Run("multiple tags sorted by date descending", func(t *testing.T) {
		tr := newTestRepo(t)
		hash := tr.commitFile("file.txt", "content", "A commit")

		t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		t2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
		t3 := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
		tr.createAnnotatedTag("v1.0.0", "First release", hash, t1)
		tr.createAnnotatedTag("v2.0.0", "Second release", hash, t2)
		tr.createAnnotatedTag("v3.0.0", "Third release", hash, t3)

		tags, err := tr.open().Tags()
		is.Err(t, err, nil)
		is.Equal(t, len(tags), 3)
		is.Equal(t, tags[0].Name(), "v3.0.0")
		is.Equal(t, tags[1].Name(), "v2.0.0")
		is.Equal(t, tags[2].Name(), "v1.0.0")
	})
}
