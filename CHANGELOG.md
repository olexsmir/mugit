# Changelog

## 0.3.0

### Breaking changes
- Switched to system sshd integration instead of bundling a custom SSH server.
- Changed route for raw files. From `/repo/blob/ref/file_path?raw=true` to `/repo/raw/ref/file_path`.

### Features:
- Paginate log page (150 commits per page).
- New compare refs page(`/{name}/compare/{ref1}/{ref2}`) with ahead/behind count, merge base, commits, and diff.
- RSS feeds: global(`/index.xml`) and per-repo(`/{name}/feed/`).
- Breadcrumbs on file content and file tree pages.
- Last commit info on tree page(per-file) and file content page.
- Show both authored and committed timestamps on commit page when they differ.
- Highlight selected line on file content page with `#L{N}` anchor links.
- Show remote urls and mirroring data on empty repos.
- Improved markdown README rendering with better typography, code blocks, callouts, and dark mode support.
- Mirror status shows last sync time(when changes were fetched) and last checked time(when checked, even without changes).
- Run `pre-receive`, `update`, `post-receive`, `post-update` hooks.
- Accept gzip-encoded HTTP requests for `git upload-pack`.
- **ssh:**
  - Support `git-upload-archive` over SSH.
  - Automatically initialize repository on first push.
  - Show modt on ssh connections (push, clone, `ssh -T`)
- **cli:**
  - `mugit repo new repo --description <desc>` sets repository description on creation.
  - `mugit repo new repo --private` creates a private repository.
  - `mugit repo new repo --mirror <url>` creates a mirror and performs initial sync.
  - `mugit repo set-default <repo.git> <branch>` changes repo default branch.
  - `mugit repo sync <repo.git>` triggers an immediate mirror sync.

### Bug fixes:
- Allow downloading only valid and existing refs.
- Support refs with special characters in names (e.g. `/` or `#`).
- Fix diffs not rendering when viewing the first commit in a repository.

## 0.2.0

### Features
- Commit Page:
  - Show both author and committer names when they differ.
  - Redesign commit page layout with improved colors and navigation.
  - Use monospace font for commit hashes.
- Format commit timestamps as `YYYY-MM-DD HH:MM:SS TZ`.
- Hide navigation bar for empty repositories.
- Render subtree-scoped README files on the tree view.
- Markdown rendering:
  - Render images with relative links within repository.
  - :hey: Add emoji support (e.g. `:smile:`).

### Bug Fixes
- Correct MIME types for raw file downloads.
- Address cases where renamed files displayed incorrectly on the commit view page.
- Fix mirrorer failing to update HEAD on empty repositories.

## 0.1.0
- Initial release
  - CLI: create, toggle private/public repo status, and add descriptions.
  - Web UI.
  - SSH server for git pull/push operations.
  - Pull-based mirroring.
