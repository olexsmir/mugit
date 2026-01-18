package git

import (
	"fmt"
	"sort"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Thanks https://git.icyphox.sh/legit/blob/master/git/git.go

type Repo struct {
	path string
	r    *git.Repository
	h    plumbing.Hash
}

// Open opens a git repository at path. If ref is empty, HEAD is used.
func Open(path string, ref string) (*Repo, error) {
	var err error
	g := Repo{}
	g.path = path
	g.r, err = git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}

	if ref == "" {
		head, err := g.r.Head()
		if err != nil {
			return nil, fmt.Errorf("getting head of %s: %w", path, err)
		}
		g.h = head.Hash()
	} else {
		hash, err := g.r.ResolveRevision(plumbing.Revision(ref))
		if err != nil {
			return nil, fmt.Errorf("resolving rev %s for %s: %w", ref, path, err)
		}
		g.h = *hash
	}
	return &g, nil
}

func (g *Repo) Commits() ([]*object.Commit, error) {
	ci, err := g.r.Log(&git.LogOptions{From: g.h})
	if err != nil {
		return nil, fmt.Errorf("commits from ref: %w", err)
	}

	commits := []*object.Commit{}
	ci.ForEach(func(c *object.Commit) error {
		commits = append(commits, c)
		return nil
	})

	return commits, nil
}

func (g *Repo) LastCommit() (*object.Commit, error) {
	c, err := g.r.CommitObject(g.h)
	if err != nil {
		return nil, fmt.Errorf("last commit: %w", err)
	}
	return c, nil
}

func (g *Repo) FileContent(path string) (string, error) {
	c, err := g.r.CommitObject(g.h)
	if err != nil {
		return "", fmt.Errorf("commit object: %w", err)
	}

	tree, err := c.Tree()
	if err != nil {
		return "", fmt.Errorf("file tree: %w", err)
	}

	file, err := tree.File(path)
	if err != nil {
		return "", err
	}

	isbin, _ := file.IsBinary()
	if !isbin {
		return file.Contents()
	} else {
		return "Not displaying binary file", nil
	}
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

func (g *Repo) Branches() ([]*plumbing.Reference, error) {
	bi, err := g.r.Branches()
	if err != nil {
		return nil, fmt.Errorf("branch: %w", err)
	}

	branches := []*plumbing.Reference{}
	err = bi.ForEach(func(ref *plumbing.Reference) error {
		branches = append(branches, ref)
		return nil
	})
	return branches, err
}

func (g *Repo) Description() (string, error) {
	c, err := g.r.Config()
	if err != nil {
		return "", fmt.Errorf("failed to read config: %w", err)
	}

	s := c.Raw.Section("mugit")
	return s.Options.Get("description"), nil
}

func (g *Repo) IsPrivate() (bool, error) {
	c, err := g.r.Config()
	if err != nil {
		return false, fmt.Errorf("failed to read config: %w", err)
	}

	s := c.Raw.Section("mugit")
	return s.Options.Get("private") == "true", nil
}

func (g *Repo) IsGoMod() bool {
	_, err := g.FileContent("go.mod")
	return err == nil
}

func (g *Repo) FindMasterBranch(masters []string) (string, error) {
	for _, b := range masters {
		if _, err := g.r.ResolveRevision(plumbing.Revision(b)); err == nil {
			return b, nil
		}
	}
	return "", fmt.Errorf("unable to find master branch")
}
