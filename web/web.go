package web

import (
	"embed"
	"io/fs"
)

var (
	//go:embed templates/* static/*
	allFS       embed.FS
	TemplatesFS = fsSub(allFS, "templates")
	StaticFS    = fsSub(allFS, "static")
)

func fsSub(fsys fs.FS, dir string) fs.FS {
	f, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return f
}
