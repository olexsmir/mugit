package git

import (
	"testing"

	"olexsmir.xyz/x/is"
)

func TestResolveName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty string", input: "", want: ".git"},
		{name: "already suffixed", input: "myrepo.git", want: "myrepo.git"},
		{name: "no suffix", input: "myrepo", want: "myrepo.git"},
		{name: ".git.git", input: "repo.git.git", want: "repo.git.git"},
		{
			name:  "special characters",
			input: "my-awesome_project",
			want:  "my-awesome_project.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is.Equal(t, ResolveName(tt.input), tt.want)
		})
	}
}

func TestResolvePath(t *testing.T) {
	base := "/repos"
	tests := []struct {
		name string
		base string
		path string
		want string
	}{
		{"simple", base, "myrepo.git", "/repos/myrepo.git"},
		{"empty", base, "", "/repos"},                                   // FIXME: block this
		{"nested", base, "user/project.git", "/repos/user/project.git"}, // FIXME: support only one level deep paths
		{"block path traversal", base, "../etc/passwd", "/repos/etc/passwd"},
		{"block absolute path", base, "/etc/passwd", "/repos/etc/passwd"},
		{"multiple traversal attempts", base, "../../../../../../etc/passwd", "/repos/etc/passwd"},
		{"base with trailing slash", "/repos/", "myrepo.git", "/repos/myrepo.git"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := ResolvePath(tt.base, tt.path)
			is.Equal(t, tt.want, path)
			is.Err(t, err, nil)
		})
	}
}
