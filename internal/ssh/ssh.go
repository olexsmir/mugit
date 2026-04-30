package ssh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"olexsmir.xyz/mugit/internal/config"
	"olexsmir.xyz/mugit/internal/git"

	gossh "golang.org/x/crypto/ssh"
)

type Shell struct {
	cfg *config.Config

	keys []gossh.PublicKey
}

func NewShell(cfg *config.Config) (*Shell, error) {
	parsedKeys := make([]gossh.PublicKey, len(cfg.SSH.Keys))
	for i, key := range cfg.SSH.Keys {
		pkey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(key))
		if err != nil {
			return nil, err
		}
		parsedKeys[i] = pkey
	}

	return &Shell{
		cfg:  cfg,
		keys: parsedKeys,
	}, nil
}

func (s *Shell) HandleCommand(ctx context.Context, cmd string, stdin io.Reader, stdout, stderr io.Writer) error {
	gitCmd, repoName, err := s.parseCommand(cmd)
	if err != nil {
		return s.replyWithGitError(stderr, "access denied: invalid command", err)
	}

	repoPath, err := git.ResolvePath(s.cfg.Repo.Dir, git.ResolveName(repoName))
	if err != nil {
		return s.replyWithGitError(stderr, "access denied", err)
	}

	repo, err := git.Open(repoPath, "")
	if err != nil {
		if !errors.Is(err, git.ErrRepoNotFound) || gitCmd != "git-receive-pack" {
			return s.replyWithGitError(stderr, "repository not found", err)
		}

		// SSH Git clients display informational messages from stderr; stdout must remain protocol-only for git-receive-pack.
		if ierr := s.replyWithGitInfo(stderr, "auto-initializing "+repoName); ierr != nil {
			return ierr
		}

		if ierr := git.Init(repoPath); ierr != nil {
			return s.replyWithGitError(stderr, "failed to init repo", ierr)
		}

		repo, err = git.Open(repoPath, "")
		if err != nil {
			return s.replyWithGitError(stderr, "failed to open initialized repo", err)
		}
	}

	switch gitCmd {
	case "git-upload-pack":
		err = repo.UploadPack(ctx, false, "", stdin, stdout)
	case "git-upload-archive":
		err = repo.UploadArchive(ctx, stdin, stdout)
	case "git-receive-pack":
		err = repo.ReceivePack(ctx, stdin, stdout, stderr)
	default:
		msg := "access denied: invalid git command"
		return s.replyWithGitError(stderr, msg, errors.New(msg))
	}

	if err != nil {
		return err
	}

	return nil
}

func (s *Shell) AuthorizedKeys(executablePath string) string {
	var out strings.Builder
	for _, key := range s.cfg.SSH.Keys {
		fmt.Fprintf(&out, `command="%s shell",no-port-forwarding,no-X11-forwarding,no-agent-forwarding,no-pty %s`+"\n",
			executablePath, key)
	}
	return out.String()
}

var validCommands = map[string]bool{
	"git-upload-pack":    true,
	"git-upload-archive": true,
	"git-receive-pack":   true,
}

func (s *Shell) parseCommand(cmd string) (gitCmd, repoName string, err error) {
	cmdParts := strings.Fields(cmd)
	if len(cmdParts) != 2 {
		return "", "", fmt.Errorf("invalid command: expected 'git-cmd repo', got %q", cmd)
	}

	gitCmd = cmdParts[0]
	if !validCommands[gitCmd] {
		return "", "", fmt.Errorf("invalid command: disallowed command")
	}

	repoName = strings.Trim(cmdParts[1], "'\"")
	if repoName == "" {
		return "", "", fmt.Errorf("invalid command: empty repository name")
	}

	return gitCmd, repoName, nil
}

func (s *Shell) replyWithGitError(stderr io.Writer, msg string, cause error) error {
	if _, err := fmt.Fprintf(stderr, "error: %s\n", msg); err != nil {
		return err
	}

	return cause
}

func (s *Shell) replyWithGitInfo(msgOut io.Writer, msg string) error {
	_, err := fmt.Fprintf(msgOut, "info: %s\n", msg)
	return err
}
