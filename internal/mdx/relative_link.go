package mdx

import (
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var (
	repoNameKey = parser.NewContextKey()
	baseDirKey  = parser.NewContextKey()
)

func NewRelativeLinkCtx(repoName, readmePath string) parser.ParseOption {
	ctx := parser.NewContext()
	ctx.Set(repoNameKey, repoName)
	ctx.Set(baseDirKey, filepath.Dir(readmePath))
	return parser.WithContext(ctx)
}

var RelativeLink = &relativeLink{}

type relativeLink struct{}

func (e *relativeLink) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(&relinkTransformer{}, 100),
		),
	)
}

type relinkTransformer struct{}

func (t *relinkTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	repoName, _ := pc.Get(repoNameKey).(string)
	baseDir, _ := pc.Get(baseDirKey).(string)

	if repoName == "" {
		return
	}

	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		var dest *[]byte
		switch v := n.(type) {
		case *ast.Image:
			dest = &v.Destination
		case *ast.Link:
			dest = &v.Destination
		default:
			return ast.WalkContinue, nil
		}

		urlStr := string(*dest)

		// skip absolute URLs
		if strings.HasPrefix(urlStr, "http://") ||
			strings.HasPrefix(urlStr, "https://") ||
			strings.HasPrefix(urlStr, "//") ||
			strings.HasPrefix(urlStr, "#") ||
			strings.HasPrefix(urlStr, "mailto:") ||
			strings.HasPrefix(urlStr, "data:") {
			return ast.WalkContinue, nil
		}

		urlStr = strings.TrimPrefix(urlStr, "./")

		var absPath string
		if after, ok := strings.CutPrefix(urlStr, "/"); ok {
			absPath = after // abs from repo root
		} else {
			// relative to repo location
			if baseDir == "" || baseDir == "." {
				absPath = urlStr
			} else {
				absPath = path.Join(baseDir, urlStr)
			}
		}

		absPath = path.Clean(absPath)
		absPath = strings.TrimPrefix(absPath, "/")

		// FIXME:hardcoded link
		*dest = fmt.Appendf(nil, "/%s/blob/HEAD/%s?raw=true",
			url.PathEscape(repoName), absPath)

		return ast.WalkContinue, nil
	})
}
