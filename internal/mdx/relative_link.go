package mdx

import (
	"bytes"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
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
		parser.WithASTTransformers(util.Prioritized(&relinkTransformer{}, 100)),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(util.Prioritized(&rawBlockRenderer{}, 100)),
	)
}

type rawBlock struct {
	ast.BaseBlock
	data []byte
}

var rawBlockKind = ast.NewNodeKind("RawBlock")

func (r *rawBlock) Kind() ast.NodeKind   { return rawBlockKind }
func (r *rawBlock) Dump(_ []byte, _ int) {}

type rawBlockRenderer struct{}

func (r *rawBlockRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(rawBlockKind, func(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			w.Write(node.(*rawBlock).data)
		}
		return ast.WalkContinue, nil
	})
}

type relinkTransformer struct{}

func (t *relinkTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	repoName, _ := pc.Get(repoNameKey).(string)
	baseDir, _ := pc.Get(baseDirKey).(string)

	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch v := n.(type) {
		case *ast.Image:
			if rewritten := t.rewriteDest(v.Destination, repoName, baseDir); rewritten != nil {
				v.Destination = rewritten
			}
		case *ast.Link:
			if rewritten := t.rewriteDest(v.Destination, repoName, baseDir); rewritten != nil {
				v.Destination = rewritten
			}
		case *ast.RawHTML:
			var buf []byte
			for i := 0; i < v.Segments.Len(); i++ {
				buf = append(buf, reader.Value(v.Segments.At(i))...)
			}
			updated := t.rewriteSrc(buf, repoName, baseDir)
			if !bytes.Equal(updated, buf) {
				str := ast.NewString(updated)
				str.SetRaw(true)
				n.Parent().ReplaceChild(n.Parent(), n, str)
			}
		case *ast.HTMLBlock:
			var buf []byte
			for i := 0; i < v.Lines().Len(); i++ {
				buf = append(buf, reader.Value(v.Lines().At(i))...)
			}
			updated := t.rewriteSrc(buf, repoName, baseDir)
			if !bytes.Equal(updated, buf) {
				n.Parent().ReplaceChild(n.Parent(), n, &rawBlock{data: updated})
			}
		}

		return ast.WalkContinue, nil
	})
}

func (t *relinkTransformer) rewriteDest(dest []byte, repoName, baseDir string) []byte {
	urlStr := string(dest)
	if strings.HasPrefix(urlStr, "http://") ||
		strings.HasPrefix(urlStr, "https://") ||
		strings.HasPrefix(urlStr, "//") ||
		strings.HasPrefix(urlStr, "#") ||
		strings.HasPrefix(urlStr, "mailto:") ||
		strings.HasPrefix(urlStr, "data:") {
		return nil
	}
	urlStr = strings.TrimPrefix(urlStr, "./")

	var absPath string
	if after, ok := strings.CutPrefix(urlStr, "/"); ok {
		absPath = after
	} else {
		if baseDir == "" || baseDir == "." {
			absPath = urlStr
		} else {
			absPath = path.Join(baseDir, urlStr)
		}
	}

	absPath = strings.TrimPrefix(path.Clean(absPath), "/")

	// FIXME: hardcoded ref and link
	return fmt.Appendf(nil, "/%s/blob/HEAD/%s?raw=true", url.PathEscape(repoName), absPath)
}

var imgSrcRe = regexp.MustCompile(`src="([^"]*)"`)

func (t *relinkTransformer) rewriteSrc(buf []byte, repoName, baseDir string) []byte {
	return imgSrcRe.ReplaceAllFunc(buf, func(match []byte) []byte {
		src := imgSrcRe.FindSubmatch(match)[1]
		rewritten := t.rewriteDest(src, repoName, baseDir)
		if rewritten == nil {
			return match
		}
		return append([]byte(`src="`), append(rewritten, '"')...)
	})
}
