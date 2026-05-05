package git

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"olexsmir.xyz/x/is"
)

func TestRepo_SetupHooks(t *testing.T) {
	t.Run("creates hook delegates and hook directories", func(t *testing.T) {
		repoPath := filepath.Join(t.TempDir(), "hooks.git")
		is.Err(t, Init(repoPath), nil)

		for _, hook := range serverHookNames {
			delegatePath := filepath.Join(repoPath, "hooks", hook)
			info, err := os.Stat(delegatePath)
			is.Err(t, err, nil)
			is.Equal(t, info.Mode()&0o111 != 0, true)

			hookDir := filepath.Join(repoPath, "hooks", hook+".d")
			dirInfo, err := os.Stat(hookDir)
			is.Err(t, err, nil)
			is.Equal(t, dirInfo.IsDir(), true)
		}
	})

	t.Run("is idempotent", func(t *testing.T) {
		repoPath := filepath.Join(t.TempDir(), "hooks.git")
		is.Err(t, Init(repoPath), nil)

		repo, err := Open(repoPath, "")
		is.Err(t, err, nil)

		is.Err(t, repo.SetupHooks(), nil)
		is.Err(t, repo.SetupHooks(), nil)
	})

	t.Run("keeps custom scripts in hook directories", func(t *testing.T) {
		repoPath := filepath.Join(t.TempDir(), "hooks.git")
		is.Err(t, Init(repoPath), nil)

		customHook := filepath.Join(repoPath, "hooks", "pre-receive.d", "90-custom.sh")
		is.Err(t, os.WriteFile(customHook, []byte("#!/bin/sh\necho ok\n"), 0o755), nil)

		repo, err := Open(repoPath, "")
		is.Err(t, err, nil)
		is.Err(t, repo.SetupHooks(), nil)

		data, err := os.ReadFile(customHook)
		is.Err(t, err, nil)
		is.Equal(t, string(data), "#!/bin/sh\necho ok\n")
	})

	t.Run("delegate forwards stdin and args to hook scripts", func(t *testing.T) {
		repoPath := filepath.Join(t.TempDir(), "hooks.git")
		is.Err(t, Init(repoPath), nil)

		delegatePath := filepath.Join(repoPath, "hooks", "update")
		delegate, err := os.ReadFile(delegatePath)
		is.Err(t, err, nil)
		firstLine, _, _ := strings.Cut(string(delegate), "\n")
		is.Equal(t, strings.HasPrefix(firstLine, "#!"), true)
		is.Equal(t, strings.Contains(firstLine, "bash"), true)
		is.Equal(t, strings.Contains(firstLine, "/usr/bin/env"), false)
		is.Equal(t, bytes.Contains(delegate, []byte(`echo "${data}" | "${hook}" "$@"`)), true)
	})
}
