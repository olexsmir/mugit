package markdown

import (
	"bytes"
	"path/filepath"

	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	callout "gitlab.com/staticnoise/goldmark-callout"
)

var (
	repoNameKey = parser.NewContextKey()
	repoRefKey  = parser.NewContextKey()
	baseDirKey  = parser.NewContextKey()

	markdown = goldmark.New(
		goldmark.WithRendererOptions(html.WithUnsafe()),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithExtensions(
			extension.GFM,
			emoji.Emoji,
			callout.CalloutExtention,
			&relativeLink{},
		))
)

func Render(repoName, repoRef, readmePath, readmeSource string) (string, error) {
	ctx := parser.NewContext()
	ctx.Set(repoNameKey, repoName)
	ctx.Set(repoRefKey, repoRef)
	ctx.Set(baseDirKey, filepath.Dir(readmePath))
	parserOpts := parser.WithContext(ctx)

	var buf bytes.Buffer
	if err := markdown.Convert([]byte(readmeSource), &buf, parserOpts); err != nil {
		return "", err
	}
	return buf.String(), nil
}
