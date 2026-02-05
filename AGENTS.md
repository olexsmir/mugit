# mugit

This is a Go-based Git repository web interface. Use these guidelines when working on this codebase.


## Project Structure

```
internal/
├── cli/             # CLI command handling
├── config/          # Configuration management (YAML)
├── git/             # Git operations
│   └── gitservice/  # git upload-pack and git receive-pack implementation
├── handlers/        # Web interface and Git HTTP protocol handlers
├── humanize/        # Time formatting utilities
├── mirror/          # Repository mirroring worker
├── ssh/             # SSH Git server
└── web/             # All things web
```

## Key Dependencies

- `github.com/urfave/cli/v3` - CLI framework
- `github.com/go-git/go-git/v5` - Pure Go Git library
- `github.com/gliderlabs/ssh` - SSH server
- `github.com/yuin/goldmark` - Markdown rendering
- `github.com/cyphar/filepath-securejoin` - Secure path joining
- `olexsmir.xyz/x/is` - Test assertions

## Security

- Always use `securejoin.SecureJoin()` when constructing filesystem paths from user input
- Check `repo.IsPrivate()` before serving public repository content
