package git

import (
	"fmt"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type TextFragment struct {
	Header string
	Lines  []gitdiff.Line
}

type Diff struct {
	Name struct {
		Old string
		New string
	}
	TextFragments []TextFragment
	IsBinary      bool
	IsNew         bool
	IsDelete      bool
}

type NiceDiff struct {
	Diff   []Diff
	Commit struct {
		Message string
		Author  object.Signature
		This    string
		Parent  string
	}
	Stat struct {
		FilesChanged int
		Insertions   int
		Deletions    int
	}
}

func (g *Repo) Diff() (*NiceDiff, error) {
	c, err := g.r.CommitObject(g.h)
	if err != nil {
		return nil, fmt.Errorf("commit object: %w", err)
	}

	patch, parent, err := g.getPatch(c)
	if err != nil {
		return nil, err
	}

	diffs, _, err := gitdiff.Parse(strings.NewReader(patch.String()))
	if err != nil {
		return nil, fmt.Errorf("parsing diff: %w", err)
	}

	nd := NiceDiff{}
	nd.Commit.Message = c.Message
	nd.Commit.Author = c.Author
	nd.Commit.This = c.Hash.String()
	nd.Commit.Parent = getParentHash(parent)
	nd.Stat.FilesChanged = len(diffs)

	nd.Diff = make([]Diff, len(diffs))
	for i, d := range diffs {
		diff := &nd.Diff[i]
		diff.Name.New = d.NewName
		diff.Name.Old = d.OldName
		diff.IsBinary = d.IsBinary
		diff.IsNew = d.IsNew
		diff.IsDelete = d.IsDelete

		for _, tf := range d.TextFragments {
			diff.TextFragments = append(diff.TextFragments, TextFragment{
				Header: tf.Header(),
				Lines:  tf.Lines,
			})
			for _, l := range tf.Lines {
				switch l.Op {
				case gitdiff.OpAdd:
					nd.Stat.Insertions += 1
				case gitdiff.OpDelete:
					nd.Stat.Deletions += 1
				}
			}
		}
	}
	return &nd, nil
}

func (g *Repo) getPatch(c *object.Commit) (*object.Patch, *object.Commit, error) {
	commitTree, err := c.Tree()
	if err != nil {
		return nil, nil, err
	}

	var parentTree *object.Tree
	var parent *object.Commit

	if c.NumParents() != 0 {
		parent, err = c.Parents().Next()
		if err != nil {
			return nil, nil, err
		}
		parentTree, err = parent.Tree()
		if err != nil {
			return nil, nil, err
		}
	} else {
		parentTree = &object.Tree{}
	}

	patch, err := parentTree.Patch(commitTree)
	if err != nil {
		return nil, nil, fmt.Errorf("patch: %w", err)
	}

	return patch, parent, nil
}

func getParentHash(parent *object.Commit) string {
	if parent == nil || parent.Hash.IsZero() {
		return ""
	}
	return parent.Hash.String()
}
