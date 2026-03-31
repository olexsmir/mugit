package git

import (
	"testing"

	"olexsmir.xyz/x/is"
)

func TestIsValidRef(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want bool
	}{
		{name: "simple branch", ref: "main", want: true},
		{name: "branch with slash", ref: "feature/new-thing", want: true},
		{name: "version tag", ref: "v1.2.3", want: true},
		{name: "short hash", ref: "abc123d", want: true},
		{name: "full hash", ref: "abc123def456789abc123def456789abc123def4", want: true},
		{name: "refs/heads path", ref: "refs/heads/main", want: true},
		{name: "refs/tags path", ref: "refs/tags/v1.0.0", want: true},
		{name: "branch with underscore", ref: "feature_branch", want: true},
		{name: "branch with dot", ref: "release.1.0", want: true},
		{name: "branch with hyphen", ref: "bug-fix", want: true},

		// security sensitive
		{name: "empty string", ref: "", want: false},
		{name: "double dot traversal", ref: "..", want: false},
		{name: "path traversal start", ref: "../etc/passwd", want: false},
		{name: "path traversal middle", ref: "refs/../../../etc/passwd", want: false},
		{name: "double dot in path", ref: "feature/..secret", want: false},

		// invalid characters
		{name: "space in name", ref: "my branch", want: false},
		{name: "newline injection", ref: "main\nmalicious", want: false},
		{name: "null byte", ref: "main\x00malicious", want: false},
		{name: "shell metachar semicolon", ref: "main;rm -rf", want: false},
		{name: "shell metachar backtick", ref: "main`whoami`", want: false},
		{name: "shell metachar dollar", ref: "main$PATH", want: false},
		{name: "shell metachar pipe", ref: "main|cat", want: false},
		{name: "shell metachar ampersand", ref: "main&id", want: false},
		{name: "single quote", ref: "main'test", want: false},
		{name: "double quote", ref: "main\"test", want: false},
		{name: "tilde", ref: "~root", want: false},
		{name: "asterisk", ref: "main*", want: false},
		{name: "question mark", ref: "main?", want: false},
		{name: "brackets", ref: "main[0]", want: false},
		{name: "parentheses", ref: "main()", want: false},
		{name: "hash", ref: "main#comment", want: false},
		{name: "percent", ref: "main%20test", want: false},
		{name: "caret", ref: "main^", want: false},
		{name: "at sign", ref: "main@{0}", want: false},
		{name: "exclamation", ref: "main!", want: false},
		{name: "backslash", ref: "main\\test", want: false},
		{name: "colon", ref: "main:test", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is.Equal(t, isValidRef(tt.ref), tt.want)
		})
	}
}
