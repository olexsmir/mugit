package git

import (
	"fmt"
	"sort"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type TagList struct {
	refs []*TagReference
	r    *git.Repository
}

// TagReference represents both lightweight and annotated tags.
// Lightweight tags contain only a reference.
// Annotated tags contain both a reference and tag metadata.
type TagReference struct {
	ref *plumbing.Reference
	tag *object.Tag
}

func (t *TagReference) Name() string {
	return t.ref.Name().Short()
}

func (t *TagReference) Message() string {
	if t.tag != nil {
		return t.tag.Message
	}
	return ""
}

func (g *Repo) Tags() ([]*TagReference, error) {
	iter, err := g.r.Tags()
	if err != nil {
		return nil, fmt.Errorf("tag objects: %w", err)
	}

	tags := make([]*TagReference, 0)
	if err := iter.ForEach(func(ref *plumbing.Reference) error {
		obj, err := g.r.TagObject(ref.Hash())
		switch err {
		case nil:
			tags = append(tags, &TagReference{
				ref: ref,
				tag: obj,
			})
		case plumbing.ErrObjectNotFound:
			tags = append(tags, &TagReference{
				ref: ref,
			})
		default:
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	tagList := &TagList{r: g.r, refs: tags}
	sort.Sort(tagList)

	return tags, nil
}

func (t *TagList) Len() int {
	return len(t.refs)
}

func (t *TagList) Swap(i, j int) {
	t.refs[i], t.refs[j] = t.refs[j], t.refs[i]
}

// Less sorting tags in reverse chronological order
func (t *TagList) Less(i, j int) bool {
	return t.getDate(i).After(t.getDate(j))
}

func (t *TagList) getDate(i int) time.Time {
	if t.refs[i].tag != nil {
		return t.refs[i].tag.Tagger.When
	}
	c, err := t.r.CommitObject(t.refs[i].ref.Hash())
	if err != nil {
		return time.Now()
	}
	return c.Committer.When
}
