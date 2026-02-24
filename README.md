# mugit

A lightweight, self-hosted Git server that your cow will love.

[See it in action!](https://git.olexsmir.xyz)

## Features

- Web interface — browse repositories, view commits, files, and diffs (no javascript required)
- Git Smart HTTP — clone over HTTPS (use SSH for pushing)
- Git over SSH — push and clone over SSH
- Mirroring — automatically mirror repositories (supports GitHub authentication)
- Private repositories — repos accessible only via SSH
- CLI — command-line for managing your repositories

## Quick install & deploy

```sh
git clone https://git.olexsmir.xyz/mugit.git
cd mugit
go build

# or
go install github.com/olexsmir/mugit@latest
```

For nixos you can use our flake, see [my config](https://git.olexsmir.xyz/dotfiles/blob/master/nix/modules/mugit.nix) for reference.

Start the server:

```sh
# start server with default config lookup
mugit serve

# start with a custom config path
mugit -c /path/to/config.yaml serve
```


## Configuration

mugit uses YAML for configuration. By default the server looks for a configuration file in this order (override with `-c` / `--config`):
1. `./config.yaml`
2. `/etc/mugit.yaml`
3. `/var/lib/mugit/config.yaml`


Durations follow Go's duration syntax (examples: `1h`, `30m`, `5s`). See: https://pkg.go.dev/time#ParseDuration

Minimal configuration example:

```yaml
meta:
  host: git.olexsmir.xyz

repo:
  dir: /var/lib/mugit
```

Full example:

```yaml
server:
  host: 0.0.0.0 # bind address (0.0.0.0 = all interfaces)
  port: 5555    # HTTP port (defaults to 8080 when omitted)

meta:
  title: "My Git Server"    # site title shown on index page
  description: "A place for my projects"
  host: git.example.com     # used for clone URLs and go-import meta tag

repo:
  dir: /var/lib/mugit   # directory with repositories
  # Default README filenames (applied when omitted):
  readmes:
    - README.md
    - readme.md
    - README.html
    - readme.html
    - README.txt
    - readme.txt
    - readme
  # Default branch names considered the repository 'master' (applied when omitted):
  masters:
    - master
    - main

# ssh: push/clone over SSH
ssh:
  enable: true
  port: 2222  # SSH port (default 2222)
  host_key: /var/lib/mugit/host   # path to SSH host key (generate with ssh-keygen)
  # Only these public keys can access private repos and push to others.
  keys:
    - ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA......
    - ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA......

# mirror: automatic mirrors of external repositories
mirror:
  enable: true
  interval: 1h  # sync frequency
  # Tokens can be provided directly, or read from environment/file:
  # - literal: "ghp_xxxxxxxxxxxx"
  # - from env: "${env:GITHUB_TOKEN}" (will read $GITHUB_TOKEN)
  # - from file: "${file:/abs/path/to/token.txt}"
  github_token: "${env:GITHUB_TOKEN}"

cache:
  home_page: 5m   # cache index/home page
  readme: 1m      # cache rendered README per repo
  diff: 15m       # cache computed diffs
```

## CLI

```sh
# start server
mugit serve

# create new public repository
mugit repo new myproject

# create new private repository
mugit repo new --private myproject

# create a mirror of an external repository
mugit repo new myproject --mirror https://codeberg.org/user/repo
mugit repo new myproject --private --mirror https://github.com/user/repo

# toggle repository visibility
mugit repo private myproject

# show and set repository description
mugit repo description myproject
mugit repo description myproject "My awesome project"
```

## License

mugit is licensed under the MIT License.
