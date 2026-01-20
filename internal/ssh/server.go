package ssh

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"

	"github.com/gliderlabs/ssh"
	"olexsmir.xyz/mugit/internal/config"
	"olexsmir.xyz/mugit/internal/git"
	"olexsmir.xyz/mugit/internal/git/gitservice"

	gossh "golang.org/x/crypto/ssh"
)

type authorizedKeyType string

const authorizedKey authorizedKeyType = "authorized"

type Server struct {
	c        *config.Config
	authKeys []gossh.PublicKey
}

func NewServer(cfg *config.Config) *Server {
	return &Server{
		c:        cfg,
		authKeys: []gossh.PublicKey{},
	}
}

func (s *Server) Start() error {
	if err := s.parseAuthKeys(); err != nil {
		return err
	}

	srv := &ssh.Server{
		Addr:             ":" + strconv.Itoa(s.c.SSH.Port),
		Handler:          s.handler,
		PublicKeyHandler: s.authhandler,
	}
	srv.SetOption(ssh.HostKeyFile(s.c.SSH.HostKey)) // TODO: validate `gossh.ParsePrivateKey`
	return srv.ListenAndServe()
}

func (s *Server) authhandler(ctx ssh.Context, key ssh.PublicKey) bool {
	fingerprint := gossh.FingerprintSHA256(key)
	if ctx.User() != "git" {
		slog.Info("non git ssh request", "user", ctx.User(), "fingerprint", fingerprint)
		return false
	}

	slog.Info("ssh request", "fingerprint", fingerprint)

	authorized := false
	for _, authKey := range s.authKeys {
		if ssh.KeysEqual(key, authKey) {
			authorized = true
			break
		}
	}

	ctx.SetValue(authorizedKey, authorized)
	return true
}

func (s *Server) handler(sess ssh.Session) {
	authorized := sess.Context().Value(authorizedKey).(bool)

	cmd := sess.Command()
	if len(cmd) < 2 {
		fmt.Fprintln(sess, "No command provided")
		sess.Exit(1)
		return
	}

	gitCmd := cmd[0]
	repoPath := cmd[1]

	repoPath = filepath.Join(s.c.Repo.Dir, filepath.Clean(repoPath))
	_, err := git.Open(repoPath, "")
	if err != nil {
		slog.Error("ssh: failed to open repo", "err", err)
		s.repoNotFound(sess)
		return
	}

	switch gitCmd {
	case "git-upload-pack":
		if err := gitservice.UploadPack(repoPath, false, sess, sess); err != nil {
			s.error(sess, err)
			return
		}
		sess.Exit(0)
	case "git-receive-pack":
		if !authorized {
			s.unauthorized(sess)
			return
		}

		if err := gitservice.ReceivePack(repoPath, sess, sess, sess.Stderr()); err != nil {
			s.error(sess, err)
			return
		}
		sess.Exit(0)

	default:
		slog.Error("ssh unsupported command", "cmd", cmd)
		gitservice.PackError(sess, "Unsupported command.")
		sess.Exit(1)
	}
}

func (s *Server) parseAuthKeys() error {
	parsedKeys := make([]gossh.PublicKey, len(s.c.SSH.Keys))
	for i, key := range s.c.SSH.Keys {
		pkey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(key))
		if err != nil {
			return err
		}
		parsedKeys[i] = pkey
	}
	s.authKeys = parsedKeys
	return nil
}

func (s *Server) repoNotFound(sess ssh.Session) {
	gitservice.PackError(sess, "Repository not found.")
	sess.Exit(1)
}

func (s *Server) unauthorized(sess ssh.Session) {
	gitservice.PackError(sess, "You are not authorized to push to this repository.")
	sess.Exit(1)
}

func (s *Server) error(sess ssh.Session, err error) {
	slog.Error("error on ssh side", "err", err)
	gitservice.PackError(sess, "Unexpected server error.")
	sess.Exit(1)
}
