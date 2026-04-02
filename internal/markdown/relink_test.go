package markdown

import (
	"testing"

	"olexsmir.xyz/x/is"
)

func TestIsAbsoluteURL(t *testing.T) {
	tests := []struct {
		name string
		link string
		want bool
	}{
		// absolute URLs
		{name: "https url", link: "https://example.com", want: true},
		{name: "http url", link: "http://example.com", want: true},
		{name: "https with path", link: "https://example.com/path/to/file", want: true},
		{name: "protocol relative", link: "//example.com/path", want: true},
		{name: "anchor link", link: "#section", want: true},
		{name: "anchor with id", link: "#my-heading-id", want: true},
		{name: "ftp scheme", link: "ftp://files.example.com", want: true},
		{name: "mailto scheme", link: "mailto:user@example.com", want: true},
		{name: "data uri", link: "data:image/png;base64,abc123", want: true},

		// relative URLs
		{name: "relative path", link: "path/to/file.md"},
		{name: "relative with dot", link: "./relative/path"},
		{name: "parent directory", link: "../other/file.md"},
		{name: "absolute path no scheme", link: "/absolute/path"},
		{name: "just filename", link: "README.md"},
		{name: "image file", link: "images/logo.png"},
		{name: "empty string", link: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is.Equal(t, isAbsoluteURL(tt.link), tt.want)
		})
	}
}

func TestRelLinkTransformer_imageFromRepo(t *testing.T) {
	tests := []struct {
		name     string
		repoName string
		repoRef  string
		baseDir  string
		dst      string
		want     string
	}{
		{
			name:     "simple image at root",
			repoName: "myrepo",
			repoRef:  "main",
			baseDir:  "",
			dst:      "logo.png",
			want:     "/myrepo/raw/main/logo.png",
		},
		{
			name:     "image in subdirectory",
			repoName: "myrepo",
			repoRef:  "main",
			baseDir:  "assets/docs",
			dst:      "images/diagram.png",
			want:     "/myrepo/raw/main/assets/docs/images/diagram.png",
		},
		{
			name:     "absolute image path",
			repoName: "myrepo",
			repoRef:  "master",
			dst:      "/assets/logo.png",
			want:     "/myrepo/raw/master/assets/logo.png",
		},
		{
			name:     "external URL unchanged",
			repoName: "myrepo",
			repoRef:  "main",
			baseDir:  "",
			dst:      "https://example.com/image.png",
			want:     "https://example.com/image.png",
		},
		{
			name:     "protocol relative URL unchanged",
			repoName: "myrepo",
			repoRef:  "main",
			baseDir:  "",
			dst:      "//cdn.example.com/image.png",
			want:     "//cdn.example.com/image.png",
		},
		{
			name:     "with version tag ref",
			repoName: "myrepo",
			repoRef:  "v1.2.3",
			baseDir:  "",
			dst:      "screenshot.png",
			want:     "/myrepo/raw/v1.2.3/screenshot.png",
		},
		{
			name:     "repo name with special chars",
			repoName: "my-repo.git",
			repoRef:  "main",
			baseDir:  "",
			dst:      "img.png",
			want:     "/my-repo.git/raw/main/img.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &relLinkTransformer{
				repoName: tt.repoName,
				repoRef:  tt.repoRef,
				baseDir:  tt.baseDir,
			}
			is.Equal(t, m.imageFromRepo(tt.dst), tt.want)
		})
	}
}

func TestRelLinkTransformer_path(t *testing.T) {
	tests := []struct {
		name    string
		baseDir string
		dst     string
		want    string
	}{
		{
			name:    "relative from root",
			baseDir: "",
			dst:     "file.md",
			want:    "file.md",
		},
		{
			name:    "relative from subdirectory",
			baseDir: "docs",
			dst:     "guide.md",
			want:    "docs/guide.md",
		},
		{
			name:    "absolute path ignores baseDir",
			baseDir: "docs",
			dst:     "/README.md",
			want:    "/README.md",
		},
		{
			name:    "nested paths",
			baseDir: "docs/api",
			dst:     "endpoints/users.md",
			want:    "docs/api/endpoints/users.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &relLinkTransformer{baseDir: tt.baseDir}
			is.Equal(t, m.path(tt.dst), tt.want)
		})
	}
}
