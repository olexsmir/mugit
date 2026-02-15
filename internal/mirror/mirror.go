package mirror

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
	"olexsmir.xyz/mugit/internal/config"
	"olexsmir.xyz/mugit/internal/git"
)

type Worker struct {
	c *config.Config
}

func NewWorker(cfg *config.Config) *Worker {
	return &Worker{
		c: cfg,
	}
}

func (w *Worker) Start(ctx context.Context) error {
	ticker := time.NewTicker(w.c.Mirror.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if err := w.mirror(ctx); err != nil {
				slog.Error("mirror sync failed", "err", err)
			}

			<-ticker.C
		}
	}
}

func (w *Worker) mirror(ctx context.Context) error {
	repos, err := w.findMirrorRepos()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	sem := semaphore.NewWeighted(10)
	errCh := make(chan error, len(repos))

	for _, repo := range repos {
		wg.Go(func() {
			if err := sem.Acquire(ctx, 1); err != nil {
				errCh <- err
				return
			}
			defer sem.Release(1)

			if err := w.syncRepo(ctx, repo); err != nil {
				errCh <- err
			}
		})
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (w *Worker) syncRepo(ctx context.Context, repo *git.Repo) error {
	name := repo.Name()
	slog.Info("mirror: sync started", "repo", name)

	remoteURL, err := repo.RemoteURL()
	if err != nil {
		slog.Error("mirror: failed to get remote url", "repo", name, "err", err)
		return err
	}

	if err := w.isSupportedRemote(remoteURL); err != nil {
		slog.Error("mirror: remote is not valid", "repo", name, "err", err)
		return err
	}

	if w.isRemoteGithub(remoteURL) && w.c.Mirror.GithubToken != "" {
		if err := repo.FetchFromGithubWithToken(ctx, w.c.Mirror.GithubToken); err != nil {
			slog.Error("mirror: fetch failed (github)", "repo", name, "err", err)
			return err
		}
	} else {
		if err := repo.Fetch(ctx); err != nil {
			slog.Error("mirror: fetch failed", "repo", name, "err", err)
			return err
		}
	}

	if err := repo.SetLastSync(time.Now()); err != nil {
		slog.Error("mirror: failed to set last sync time", "repo", name, "err", err)
	}

	slog.Info("mirror: sync completed", "repo", repo.Name())
	return nil
}

func (w *Worker) findMirrorRepos() ([]*git.Repo, error) {
	dirs, err := os.ReadDir(w.c.Repo.Dir)
	if err != nil {
		return nil, err
	}

	var repos []*git.Repo
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		name := dir.Name()
		path := filepath.Join(w.c.Repo.Dir, filepath.Clean(name))
		repo, err := git.Open(path, "")
		if err != nil {
			slog.Debug("skipping non-git directory", "name", name, "err", err)
			continue
		}

		isMirror, err := repo.IsMirror()
		if err != nil {
			slog.Debug("skipping non-mirror repo", "name", name, "err", err)
			continue
		}

		if isMirror {
			repos = append(repos, repo)
		}
	}

	return repos, nil
}

func (w *Worker) isSupportedRemote(remote string) error {
	if !strings.HasPrefix(remote, "http") {
		return fmt.Errorf("only http and https remotes are supported")
	}
	return nil
}

func (w *Worker) isRemoteGithub(remoteURL string) bool {
	return strings.Contains(remoteURL, "github.com")
}
