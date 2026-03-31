package git

import (
	"testing"

	"olexsmir.xyz/x/is"
)

func TestRepo_FileTree(t *testing.T) {
	t.Run("root tree", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("README.md", "# Test", "Initial commit")
		r.commitFile("src/main.go", "package main", "Add main.go")
		r.commitFile("docs/guide.md", "# Guide", "Add guide")

		tree, err := r.open().FileTree(t.Context(), "")
		is.Err(t, err, nil)

		names := make(map[string]bool)
		for _, entry := range tree {
			names[entry.Name] = true
		}
		is.Equal(t, names["README.md"], true)
		is.Equal(t, names["src"], true)
		is.Equal(t, names["docs"], true)
	})

	t.Run("subdirectory", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("src/main.go", "package main", "Add main.go")
		r.commitFile("src/util/helper.go", "package util", "Add helper")

		tree, err := r.open().FileTree(t.Context(), "src")
		is.Err(t, err, nil)

		names := make(map[string]bool)
		for _, entry := range tree {
			names[entry.Name] = true
		}

		is.Equal(t, names["main.go"], true)
		is.Equal(t, names["util"], true)
	})

	t.Run("distinguishes files and directories", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("file.txt", "content", "Add file")
		r.commitFile("dir/nested.txt", "nested", "Add nested")

		tree, err := r.open().FileTree(t.Context(), "")
		is.Err(t, err, nil)
		for _, entry := range tree {
			switch entry.Name {
			case "file.txt":
				is.Equal(t, entry.IsFile, true)
				is.Equal(t, entry.Commit.Message, "Add file")
			case "dir":
				is.Equal(t, entry.IsFile, false)
				is.Equal(t, entry.Commit.Message, "Add nested")
			}
		}
	})

	t.Run("includes file sizes", func(t *testing.T) {
		r := newTestRepo(t)
		content := "Hello, World!"
		r.commitFile("hello.txt", content, "Add hello")

		tree, err := r.open().FileTree(t.Context(), "")
		is.Err(t, err, nil)
		for _, entry := range tree {
			if entry.Name == "hello.txt" {
				is.Equal(t, entry.Size, int64(len(content)))
				is.Equal(t, entry.Commit.Message, "Add hello")
			}
		}
	})
}

func TestRepo_FileContent(t *testing.T) {
	t.Run("returns text file content", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("hello.txt", "Hello, World!", "Add hello")

		fc, err := r.open().FileContent("hello.txt")
		is.Err(t, err, nil)
		is.Equal(t, fc.String(), "Hello, World!")
		is.Equal(t, fc.IsBinary, false)
		is.Equal(t, fc.IsImage, false)
	})

	t.Run("returns file in subdirectory", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("lua/hello.lua", `vim.print "hi"`, "add stuff")

		fc, err := r.open().FileContent("lua/hello.lua")
		is.Err(t, err, nil)
		is.Equal(t, fc.String(), `vim.print "hi"`)
	})

	t.Run("returns ErrFileNotFound for missing file", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("dummy.txt", "dummy", "Initial commit")

		_, err := r.open().FileContent("nonexistent.txt")
		is.Err(t, err, ErrFileNotFound)
	})

	t.Run("detects mime type from extension", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("style.css", "body { color: red; }", "Add css")
		r.commitFile("script.js", "console.log('hi')", "Add js")

		repo := r.open()
		css, err := repo.FileContent("style.css")
		is.Equal(t, css.Mime, "text/css; charset=utf-8")
		is.Err(t, err, nil)

		js, err := repo.FileContent("script.js")
		is.Equal(t, js.Mime, "text/javascript; charset=utf-8")
		is.Err(t, err, nil)
	})

	t.Run("defaults to text/plain for unknown extension", func(t *testing.T) {
		r := newTestRepo(t)
		r.commitFile("data.mugitunknown", "some data", "Add data")

		fc, err := r.open().FileContent("data.mugitunknown")
		is.Equal(t, fc.Mime, "text/plain")
		is.Err(t, err, nil)
	})
}

func TestFileContent_String(t *testing.T) {
	t.Run("returns content for text", func(t *testing.T) {
		is.Equal(t, (&FileContent{
			Content:  []byte("hello"),
			IsBinary: false,
			IsImage:  false,
		}).String(), "hello")
	})

	t.Run("returns empty for binary and images", func(t *testing.T) {
		is.Equal(t, (&FileContent{
			Content:  []byte("binary-data"),
			IsBinary: true,
			IsImage:  false,
		}).String(), "")

		is.Equal(t, (&FileContent{
			Content:  []byte("image data"),
			IsBinary: false,
			IsImage:  true,
		}).String(), "")
	})
}
