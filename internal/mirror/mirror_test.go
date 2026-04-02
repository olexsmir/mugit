package mirror

import (
	"testing"

	"olexsmir.xyz/x/is"
)

func TestIsRemoteSupported(t *testing.T) {
	tests := []struct {
		name    string
		remote  string
		wantErr bool
	}{
		// supported
		{name: "https url", remote: "https://github.com/user/repo.git"},
		{name: "http url", remote: "http://example.com/repo.git"},
		{name: "https without .git", remote: "https://github.com/user/repo"},

		// unsupported
		{name: "ssh url", remote: "git@github.com:user/repo.git", wantErr: true},
		{name: "git protocol", remote: "git://github.com/user/repo.git", wantErr: true},
		{name: "local path", remote: "/path/to/repo", wantErr: true},
		{name: "relative path", remote: "../other-repo", wantErr: true},
		{name: "file protocol", remote: "file:///path/to/repo", wantErr: true},
		{name: "empty string", remote: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsRemoteSupported(tt.remote)
			if tt.wantErr {
				is.Err(t, err, "only http and https")
			} else {
				is.Err(t, err, nil)
			}
		})
	}
}

func TestIsGithubRemote(t *testing.T) {
	tests := []struct {
		name   string
		remote string
		want   bool
	}{
		{name: "github https", remote: "https://github.com/user/repo.git", want: true},
		{name: "github http", remote: "http://github.com/user/repo", want: true},
		{name: "github enterprise", remote: "https://github.mycompany.com/user/repo", want: false},
		{name: "raw github", remote: "https://raw.github.com/user/repo/file", want: true},

		{name: "gitlab", remote: "https://gitlab.com/user/repo.git", want: false},
		{name: "bitbucket", remote: "https://bitbucket.org/user/repo.git", want: false},
		{name: "generic git server", remote: "https://git.example.com/repo.git", want: false},
		{name: "empty url", remote: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is.Equal(t, IsGithubRemote(tt.remote), tt.want)
		})
	}
}
