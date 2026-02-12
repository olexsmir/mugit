package ssh

import (
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/gliderlabs/ssh"
	"olexsmir.xyz/mugit/internal/config"
	"olexsmir.xyz/mugit/internal/git"
	"olexsmir.xyz/mugit/internal/git/gitx"

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

	if err := srv.SetOption(ssh.HostKeyFile(s.c.SSH.HostKey)); err != nil {
		// TODO: validate `gossh.ParsePrivateKey`
		return err
	}

	return srv.ListenAndServe()
}

func (s *Server) authhandler(ctx ssh.Context, key ssh.PublicKey) bool {
	fingerprint := gossh.FingerprintSHA256(key)
	if ctx.User() != "git" {
		slog.Info("non git ssh request", "user", ctx.User(), "fingerprint", fingerprint)
		return false
	}

	slog.Info("ssh request", "fingerprint", fingerprint)
	authorized := slices.ContainsFunc(s.authKeys, func(i gossh.PublicKey) bool {
		return ssh.KeysEqual(key, i)
	})
	ctx.SetValue(authorizedKey, authorized)
	return true
}

func (s *Server) handler(sess ssh.Session) {
	ctx := sess.Context()
	authorized := sess.Context().Value(authorizedKey).(bool)

	cmd := sess.Command()
	if len(cmd) < 2 {
		fmt.Fprintln(sess, "No command provided")
		sess.Exit(1)
		return
	}

	gitCmd := cmd[0]
	rawRepoPath := cmd[1]
	normalizedRepoName := normalizeRepoName(rawRepoPath)
	repoPath := repoNameToPath(normalizedRepoName)

	fullPath, err := securejoin.SecureJoin(s.c.Repo.Dir, repoPath)
	if err != nil {
		slog.Error("ssh: invalid path", "err", err)
		s.repoNotFound(sess)
		return
	}

	repo, err := git.Open(fullPath, "")
	if err != nil {
		slog.Error("ssh: failed to open repo", "err", err)
		s.repoNotFound(sess)
		return
	}

	switch gitCmd {
	case "git-upload-pack":
		isPrivate, err := repo.IsPrivate()
		if err != nil {
			s.error(sess, err)
			return
		}

		if isPrivate && !authorized {
			s.repoNotFound(sess)
			return
		}

		if err := gitx.UploadPack(ctx, fullPath, false, sess, sess); err != nil {
			s.error(sess, err)
			return
		}
		sess.Exit(0)
	case "git-receive-pack":
		if !authorized {
			s.unauthorized(sess)
			return
		}

		if err := gitx.ReceivePack(ctx, fullPath, sess, sess, sess.Stderr()); err != nil {
			s.error(sess, err)
			return
		}
		sess.Exit(0)

	default:
		slog.Error("ssh unsupported command", "cmd", cmd)
		gitx.PackError(sess, "Unsupported command.")
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
	gitx.PackError(sess, "Repository not found.")
	sess.Exit(1)
}

func (s *Server) unauthorized(sess ssh.Session) {
	gitx.PackError(sess, "You are not authorized to push to this repository.")
	sess.Exit(1)
}

func (s *Server) error(sess ssh.Session, err error) {
	slog.Error("error on ssh side", "err", err)
	gitx.PackError(sess, "Unexpected server error.")
	sess.Exit(1)
}

func repoNameToPath(name string) string { return name + ".git" }
func normalizeRepoName(name string) string {
	return strings.TrimSuffix(name, ".git")
}
