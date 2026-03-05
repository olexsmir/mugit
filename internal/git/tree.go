package git

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/object"
)

type NiceTree struct {
	IsFile bool
	Name   string
	Commit *Commit
	Mode   string
	Size   int64
}

func (g *Repo) makeNiceTree(t *object.Tree, parent string) []NiceTree {
	var nts []NiceTree

	cms, err := g.lastCommitForFilesInTree(t, parent)
	if err != nil {
		return nts
	}

	for _, e := range t.Entries {
		mode, _ := e.Mode.ToOSFileMode()
		sz, _ := t.Size(e.Name)
		nts = append(nts, NiceTree{
			Name:   e.Name,
			Mode:   mode.String(),
			IsFile: e.Mode.IsFile(),
			Commit: cms[e.Name],
			Size:   sz,
		})
	}
	return nts
}

func (g *Repo) FileTree(path string) ([]NiceTree, error) {
	c, err := g.r.CommitObject(g.h)
	if err != nil {
		return nil, fmt.Errorf("commit object: %w", err)
	}

	tree, err := c.Tree()
	if err != nil {
		return nil, fmt.Errorf("file tree: %w", err)
	}

	var files []NiceTree
	if path == "" {
		files = g.makeNiceTree(tree, path)
	} else {
		o, err := tree.FindEntry(path)
		if err != nil {
			return nil, err
		}

		if !o.Mode.IsFile() {
			subtree, err := tree.Tree(path)
			if err != nil {
				return nil, err
			}
			files = g.makeNiceTree(subtree, path)
		}
	}

	return files, nil
}

type FileContent struct {
	IsBinary bool
	IsImage  bool
	Content  []byte
	Mime     string
	Size     int64
}

func (fc *FileContent) String() string {
	if fc.IsBinary || fc.IsImage {
		return ""
	}
	return string(fc.Content)
}

func (g *Repo) FileContent(path string) (*FileContent, error) {
	c, err := g.r.CommitObject(g.h)
	if err != nil {
		return &FileContent{}, fmt.Errorf("commit object: %w", err)
	}

	tree, err := c.Tree()
	if err != nil {
		return &FileContent{}, fmt.Errorf("file tree: %w", err)
	}

	file, err := tree.File(path)
	if err != nil {
		if errors.Is(err, object.ErrFileNotFound) {
			return &FileContent{}, ErrFileNotFound
		}
		return &FileContent{}, err
	}

	reader, err := file.Reader()
	if err != nil {
		return nil, fmt.Errorf("file reader: %w", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	isBin, _ := file.IsBinary()
	mimeType := mime.TypeByExtension(filepath.Ext(path))
	if mimeType == "" {
		mimeType = "text/plain"
		if isBin {
			mimeType = "application/octet-stream"
		}
	}

	return &FileContent{
		IsBinary: isBin,
		IsImage:  strings.HasPrefix(mimeType, "image/"),
		Content:  content,
		Mime:     mimeType,
		Size:     file.Size,
	}, nil
}
