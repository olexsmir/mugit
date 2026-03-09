package git

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type NiceTree struct {
	IsFile bool
	Name   string
	Commit *Commit
	Mode   string
	Size   int64
}

func (g *Repo) makeNiceTree(ctx context.Context, t *object.Tree, parent string) []NiceTree {
	var nts []NiceTree

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	cms, err := g.lastCommitForFilesInTree(ctx, t, parent)
	if err != nil {
		return nts
	}

	for _, e := range t.Entries {
		fpath := path.Join(parent, e.Name)
		mode, _ := e.Mode.ToOSFileMode()
		sz, _ := t.Size(e.Name)
		nts = append(nts, NiceTree{
			Commit: cms[fpath],
			Name:   e.Name,
			Mode:   mode.String(),
			IsFile: e.Mode.IsFile(),
			Size:   sz,
		})
	}
	return nts
}

func (g *Repo) FileTree(ctx context.Context, path string) ([]NiceTree, error) {
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
		files = g.makeNiceTree(ctx, tree, path)
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
			files = g.makeNiceTree(ctx, subtree, path)
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

type logCommit struct {
	Commit
	hash  plumbing.Hash
	files []string
}

func (g *Repo) lastCommitForFilesInTree(ctx context.Context, subtree *object.Tree, parent string) (map[string]*Commit, error) {
	filesToDo := make(map[string]struct{})
	filesDone := make(map[string]*Commit)
	for _, e := range subtree.Entries {
		fpath := path.Clean(path.Join(parent, e.Name))
		filesToDo[fpath] = struct{}{}
	}

	if len(filesToDo) == 0 {
		return filesDone, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pathSpec := "."
	if parent != "" {
		pathSpec = parent
	}

	output, err := g.streamingGitLog(ctx, "--pretty=format:%H,%ad,%ae,%an,%ce,%cn,%s", "--date=iso", "--name-only", "--", pathSpec)
	if err != nil {
		return nil, err
	}
	defer output.Close() // Ensure the git process is properly cleaned up

	var current logCommit
	reader := bufio.NewReader(output)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			if !current.hash.IsZero() {
				c := current.Commit
				// we have a fully parsed commit
				for _, f := range current.files {
					if _, ok := filesToDo[f]; ok {
						filesDone[f] = &c
						delete(filesToDo, f)
					}
				}

				if len(filesToDo) == 0 {
					cancel()
					break
				}

				current = logCommit{}
			}
		} else if current.hash.IsZero() {
			parts := strings.SplitN(line, ",", 7)
			if len(parts) == 7 {
				current.hash = plumbing.NewHash(parts[0])

				// NOTE: this is copy-paste of [newCommit]
				current.Hash = parts[0]
				current.HashShort = parts[0][:7]
				current.Committed, _ = time.Parse("2006-01-02 15:04:05 -0700", parts[1])
				current.AuthorEmail = parts[2]
				current.AuthorName = parts[3]
				current.CommitterEmail = parts[4]
				current.CommitterName = parts[5]
				current.Message = parts[6]
			}
		} else {
			// all ancestors along this path should also be included
			file := path.Clean(line)
			ancestors := ancestors(file)
			current.files = append(current.files, file)
			current.files = append(current.files, ancestors...)
		}

		if err == io.EOF {
			break
		}
	}

	return filesDone, nil
}

func ancestors(p string) []string {
	var ancestors []string
	for {
		p = path.Dir(p)
		if p == "." || p == "/" {
			break
		}
		ancestors = append(ancestors, p)
	}
	return ancestors
}
