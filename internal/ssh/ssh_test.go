package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	gossh "golang.org/x/crypto/ssh"
	"olexsmir.xyz/x/is"
)

func TestServer_isAuthorized(t *testing.T) {
	key1, err := rsa.GenerateKey(rand.Reader, 2048)
	is.Err(t, err, nil)
	pub1, err := gossh.NewPublicKey(&key1.PublicKey)
	is.Err(t, err, nil)

	key2, err := rsa.GenerateKey(rand.Reader, 2048)
	is.Err(t, err, nil)
	pub2, err := gossh.NewPublicKey(&key2.PublicKey)
	is.Err(t, err, nil)

	tests := []struct {
		name     string
		authKeys []gossh.PublicKey
		checkKey gossh.PublicKey
		wantAuth bool
	}{
		{
			name:     "authorized key",
			wantAuth: true,
			authKeys: []gossh.PublicKey{pub1},
			checkKey: pub1,
		},
		{
			name:     "unauthorized key",
			wantAuth: false,
			authKeys: []gossh.PublicKey{pub1},
			checkKey: pub2,
		},
		{
			name:     "empty auth keys",
			wantAuth: false,
			authKeys: []gossh.PublicKey{},
			checkKey: pub1,
		},
		{
			name:     "multiple auth keys - found",
			wantAuth: true,
			authKeys: []gossh.PublicKey{pub1, pub2},
			checkKey: pub2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{authKeys: tt.authKeys}
			got := s.isAuthorized(tt.checkKey)
			is.Equal(t, tt.wantAuth, got)
		})
	}
}
