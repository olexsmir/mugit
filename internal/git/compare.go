package git

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
)

type Compare struct {
	BaseRef, BaseHash string
	HeadRef, HeadHash string
	MergeBase         string
	Ahead             int
	Behind            int
	Commits           []*Commit
	Diff              *NiceDiff
}

func (g *Repo) Compare(baseRef, headRef string) (*Compare, error) {
	if baseRef == "" || headRef == "" {
		return nil, errors.New("base and head refs can not be empty")
	}

	baseHash, err := g.resolveRef(baseRef)
	if err != nil {
		return nil, fmt.Errorf("resolving base ref %q: %w", baseRef, err)
	}

	headHash, err := g.resolveRef(headRef)
	if err != nil {
		return nil, fmt.Errorf("resolving head ref %q: %w", headRef, err)
	}

	mergeBaseOut, err := g.mergeBase(baseHash.String(), headHash.String())
	if err != nil {
		return nil, fmt.Errorf("merge-base for %q and %q: %w", baseRef, headRef, err)
	}

	mergeBase := strings.TrimSpace(string(mergeBaseOut))
	if mergeBase == "" {
		return nil, fmt.Errorf("merge-base for %q and %q: empty output", baseRef, headRef)
	}

	countsOut, err := g.revList("--left-right", "--count", fmt.Sprintf("%s...%s", baseHash.String(), headHash.String()))
	if err != nil {
		return nil, fmt.Errorf("ahead/behind for %q and %q: %w", baseRef, headRef, err)
	}

	behind, ahead, err := parseAheadBehind(countsOut)
	if err != nil {
		return nil, err
	}

	commits, err := g.commitsInRange(baseHash, headHash)
	if err != nil {
		return nil, err
	}

	diff, err := g.diffBetween(plumbing.NewHash(mergeBase), headHash)
	if err != nil {
		return nil, err
	}

	return &Compare{
		BaseRef:   baseRef,
		HeadRef:   headRef,
		BaseHash:  baseHash.String(),
		HeadHash:  headHash.String(),
		MergeBase: mergeBase,
		Ahead:     ahead,
		Behind:    behind,
		Commits:   commits,
		Diff:      diff,
	}, nil
}

func (g *Repo) resolveRef(ref string) (plumbing.Hash, error) {
	hash, err := g.r.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return plumbing.ZeroHash, err
	}
	return *hash, nil
}

func parseAheadBehind(counts []byte) (behind, ahead int, err error) {
	fields := strings.Fields(strings.TrimSpace(string(counts)))
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("unexpected ahead/behind format: %q", counts)
	}

	behind, err = strconv.Atoi(fields[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid behind count %q: %w", fields[0], err)
	}

	ahead, err = strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid ahead count %q: %w", fields[1], err)
	}

	return behind, ahead, nil
}

func (g *Repo) commitsInRange(base, head plumbing.Hash) ([]*Commit, error) {
	out, err := g.runGitCmd("log", "--format=%H", fmt.Sprintf("%s..%s", base.String(), head.String()))
	if err != nil {
		return nil, fmt.Errorf("commits in range %s..%s: %w", base, head, err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []*Commit{}, nil
	}

	commits := make([]*Commit, 0, len(lines))
	for _, hash := range lines {
		hash = strings.TrimSpace(hash)
		if hash == "" {
			continue
		}

		c, err := g.r.CommitObject(plumbing.NewHash(hash))
		if err != nil {
			return nil, fmt.Errorf("commit object %s: %w", hash, err)
		}
		commits = append(commits, newCommit(c))
	}
	return commits, nil
}

func (g *Repo) revList(args ...string) ([]byte, error) {
	return g.runGitCmd("rev-list", args...)
}

func (g *Repo) mergeBase(args ...string) ([]byte, error) {
	return g.runGitCmd("merge-base", args...)
}
