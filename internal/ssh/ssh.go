package ssh

import (
	"fmt"
	"log/slog"
	"slices"
	"strconv"

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

	ctx.SetValue(authorizedKey, s.isAuthorized(key))
	return true
}

func (s *Server) handler(sess ssh.Session) {
	ctx := sess.Context()
	authorized := ctx.Value(authorizedKey).(bool)

	cmd := sess.Command()
	if len(cmd) < 2 {
		s.error(sess, badRequestErrMsg, nil)
		return
	}

	gitCmd := cmd[0]
	userProvidedRepoName := cmd[1]
	repoPath, err := git.ResolvePath(s.c.Repo.Dir, git.ResolveName(userProvidedRepoName))
	if err != nil {
		s.error(sess, internalServerErrMsg, err)
		return
	}

	repo, err := git.Open(repoPath, "")
	if err != nil {
		s.gitError(sess, repoNotFoundErrMsg, err)
		return
	}

	switch gitCmd {
	case "git-upload-pack":
		isPrivate, err := repo.IsPrivate()
		if err != nil {
			s.gitError(sess, badRequestErrMsg, nil)
			return
		}

		if isPrivate && !authorized {
			s.gitError(sess, badRequestErrMsg, nil)
			return
		}

		if err := gitx.UploadPack(ctx, repoPath, false, sess, sess); err != nil {
			s.gitError(sess, internalServerErrMsg, err)
			return
		}

		sess.Exit(0)

	case "git-upload-archive":
		isPrivate, err := repo.IsPrivate()
		if err != nil {
			s.gitError(sess, badRequestErrMsg, nil)
			return
		}

		if isPrivate && !authorized {
			s.gitError(sess, badRequestErrMsg, nil)
			return
		}

		if err := gitx.UploadArchive(ctx, repoPath, sess, sess); err != nil {
			s.gitError(sess, internalServerErrMsg, err)
			return
		}

		sess.Exit(0)

	case "git-receive-pack":
		if !authorized {
			s.gitError(sess, unauthorizedErrMsg, nil)
			return
		}

		if err := gitx.ReceivePack(ctx, repoPath, sess, sess, sess.Stderr()); err != nil {
			s.gitError(sess, internalServerErrMsg, err)
			return
		}

		sess.Exit(0)

	default:
		s.error(sess, badRequestErrMsg, nil)
		return
	}
}

func (s *Server) isAuthorized(iden gossh.PublicKey) bool {
	return slices.ContainsFunc(s.authKeys, func(i gossh.PublicKey) bool {
		return ssh.KeysEqual(iden, i)
	})
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

const (
	internalServerErrMsg = "internal server error\n"
	badRequestErrMsg     = "bad request\n"
	unauthorizedErrMsg   = "pushing only allowed to authorized users\n"
	repoNotFoundErrMsg   = "repository not found\n"
)

func (s *Server) error(sess ssh.Session, msg string, err error) {
	slog.Error("ssh error", "msg", msg, "err", err)
	fmt.Fprintf(sess.Stderr(), "%s", msg)
	sess.Exit(1)
}

func (s *Server) gitError(sess ssh.Session, msg string, err error) {
	slog.Error("ssh git error", "msg", msg, "err", err)
	gitx.PackError(sess, msg)
	gitx.PackFlush(sess)
	sess.Exit(1)
}
