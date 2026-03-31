package handlers

import (
	"testing"

	"olexsmir.xyz/x/is"
)

func TestBreadcrumbs(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []Breadcrumb
	}{
		{name: "empty path", path: "", want: nil},
		{
			name: "single segment",
			path: "src",
			want: []Breadcrumb{{Name: "src", Path: "src", IsLast: true}},
		},
		{
			name: "two segments",
			path: "src/main",
			want: []Breadcrumb{
				{Name: "src", Path: "src", IsLast: false},
				{Name: "main", Path: "src/main", IsLast: true},
			},
		},
		{
			name: "deep nesting",
			path: "src/internal/handlers/repo.go",
			want: []Breadcrumb{
				{Name: "src", Path: "src", IsLast: false},
				{Name: "internal", Path: "src/internal", IsLast: false},
				{Name: "handlers", Path: "src/internal/handlers", IsLast: false},
				{Name: "repo.go", Path: "src/internal/handlers/repo.go", IsLast: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is.Equal(t, Breadcrumbs(tt.path), tt.want)
		})
	}
}

func TestParseRef(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "simple ref", input: "main", want: "main"},
		{name: "url encoded slash", input: "feature%2Fnew-thing", want: "feature/new-thing"},
		{name: "url encoded spaces", input: "my%20branch", want: "my branch"},
		{name: "already decoded", input: "refs/heads/main", want: "refs/heads/main"},
		{name: "version tag", input: "v1.2.3", want: "v1.2.3"},
		{name: "hash", input: "abc123def", want: "abc123def"},
	}

	h := handlers{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is.Equal(t, h.parseRef(tt.input), tt.want)
		})
	}
}

func TestTemplate_CommitSummary(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty string", input: "", want: ""},
		{name: "single line", input: "Fix bug in handler", want: "Fix bug in handler"},
		{name: "trailing newline only", input: "Fix bug\n", want: "Fix bug"},
		{
			name:  "no blank line separator (malformed)",
			input: "Fix bug\nMore details",
			want:  "Fix bug...",
		},
		{
			name:  "proper body with blank line",
			input: "Fix bug\n\nMore details here",
			want:  "Fix bug...",
		},
		{
			name:  "multiple body paragraphs",
			input: "Fix bug\n\nMore details\n\nEven more",
			want:  "Fix bug...",
		},
		{
			name:  "trailing blank lines only",
			input: "Fix bug\n\n",
			want:  "Fix bug",
		},
		{
			name:  "with CRLF no body",
			input: "Fix bug\r\n",
			want:  "Fix bug",
		},
		{
			name:  "with CRLF and body",
			input: "Fix bug\r\n\r\nMore details",
			want:  "Fix bug...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is.Equal(t, commitSummary(tt.input), tt.want)
		})
	}
}
