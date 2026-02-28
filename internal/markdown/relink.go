package markdown

import (
	"bytes"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type relativeLink struct{}

func (e *relativeLink) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithASTTransformers(util.Prioritized(&relLinkTransformer{}, 99)))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(&rawBlockRenderer{}, 100)))
}

type relLinkTransformer struct {
	repoName string
	repoRef  string
	baseDir  string
}

func (m *relLinkTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	m.repoName, _ = pc.Get(repoNameKey).(string)
	m.repoRef, _ = pc.Get(repoRefKey).(string)
	m.baseDir, _ = pc.Get(baseDirKey).(string)

	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch n := n.(type) {
		case *ast.Link:
			m.relativeLinkTransformer(n)

		case *ast.Image:
			m.imageFromRepoTransformer(n)

		case *ast.RawHTML:
			buf := m.segmentsToBytes(n.Segments, reader)
			updated := m.rewriteSrc(buf)
			if !bytes.Equal(updated, buf) {
				s := ast.NewString(updated)
				s.SetRaw(true)
				n.Parent().ReplaceChild(n.Parent(), n, s)
			}

		case *ast.HTMLBlock:
			buf := m.segmentsToBytes(n.Lines(), reader)
			updated := m.rewriteSrc(buf)
			if !bytes.Equal(updated, buf) {
				n.Parent().ReplaceChild(n.Parent(), n, &rawBlock{data: updated})
			}
		}

		return ast.WalkContinue, nil
	})
}

func (m *relLinkTransformer) relativeLinkTransformer(link *ast.Link) {
	dst := string(link.Destination)
	if isAbsoluteURL(dst) {
		return
	}

	act := m.path(dst)
	link.Destination = []byte(path.Join("/", m.repoName, "tree", m.repoRef, act))
}

func (m *relLinkTransformer) imageFromRepoTransformer(img *ast.Image) {
	img.Destination = []byte(m.imageFromRepo(
		string(img.Destination)))
}

func (m *relLinkTransformer) imageFromRepo(dst string) string {
	if isAbsoluteURL(dst) {
		return dst
	}

	absPath := m.path(dst)
	return path.Join("/", url.PathEscape(m.repoName), "blob", m.repoRef, absPath) +
		"?raw=true"
}

func (m *relLinkTransformer) path(dst string) string {
	if path.IsAbs(dst) {
		return dst
	}
	return path.Join(m.baseDir, dst)
}

var imgSrcRe = regexp.MustCompile(`src="([^"]*)"`)

func (m *relLinkTransformer) rewriteSrc(buf []byte) []byte {
	return imgSrcRe.ReplaceAllFunc(buf, func(match []byte) []byte {
		start := bytes.IndexByte(match, '"') + 1
		end := bytes.LastIndexByte(match, '"')
		src := string(match[start:end])
		return []byte(`src="` + m.imageFromRepo(src) + `"`)
	})
}

func (m *relLinkTransformer) segmentsToBytes(segs *text.Segments, reader text.Reader) []byte {
	var buf []byte
	for i := 0; i < segs.Len(); i++ {
		buf = append(buf, reader.Value(segs.At(i))...)
	}
	return buf
}

func isAbsoluteURL(link string) bool {
	if strings.HasPrefix(link, "#") {
		return true
	}
	u, err := url.Parse(link)
	return err == nil && (u.Scheme != "" || strings.HasPrefix(link, "//"))
}

// row block

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
