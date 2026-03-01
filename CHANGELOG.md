# Changelog

## 0.3.0 (unreleased)

### Features:
- Paginate log page (150 commits per page).
- Support `git-upload-archive`.
- Better CSS for markdown readmes.

### Bug fixes:
- Allow downloading only valid and existing refs.
- Support refs with special characters in names (e.g. `/` or `#`).
- Previously when viewing first commit in repo diffs were not rendered.

## 0.2.0

### Features
- Commit Page:
  - Show both author and committer names when they differ.
  - Redesign commit page layout with improved colors and navigation.
  - Use mono font for commit hashes.
- Format commit timestamps as `YYYY-MM-DD HH:MM:SS TZ`.
- Hide navigation bar for empty repositories.
- Render subtree-scoped README files on the tree view.
- Markdown rendering:
  - Render images with relative links within repository.
  - Add emoji support :hey: (e.g. `:smile:`).

### Bug Fixes
- Correct MIME types for raw file downloads.
- Address cases where renamed files displayed incorrectly on the commit view page.
- Fix mirrorer failing to update HEAD on empty repositories.

## 0.1.0
- Initial release
  - CLI: create, toggle private/public repo status, and add descriptions.
  - Web UI.
  - SSH server for git pull/push operations.
  - Pull based mirroring.
