# mugit

A lightweight, self-hosted Git server that your cow will love.

[See it in action!](https://git.olexsmir.xyz)

## Features
- Web interface — browse repositories, view commits, files, and diffs (no javascript required).
- Git Smart HTTP — clone over HTTPS (use SSH for pushing).
- Git over SSH — push and clone repos over SSH.
- Mirroring — automatically mirror repos from other forges (supports GitHub authentication).
- Private repositories — repos accessible only via SSH
- CLI — command-line for managing your repositories

## Quick install & deploy

```sh
# get binary
git clone https://git.olexsmir.xyz/mugit.git
cd mugit
go build
# or
go install github.com/olexsmir/mugit@latest


# start server
mugit serve
```

<details>
<summary>Deploy guide</summary>

  If you're on nixos feel free to use mugit's flake. See [example config](https://git.olexsmir.xyz/dotfiles/blob/master/nix/modules/services/mugit.nix) for a full reference.

  1. Get a mugit binary.

  Download it from [github releases](https://github.com/olexsmir/mugit/releases) or build it from source:
  ```bash
  git clone https://git.olexsmir.xyz/mugit.git
  cd mugit
  go build -o /usr/local/bin/mugit
  ```

  2. Create a user for mugit, and repo for your repo.
  ```bash
  useradd -r -d /var/lib/mugit -m -s /bin/sh mugit
  mkdir -p /var/lib/mugit
  ```

  3. Configure
  ```yaml
  # file: /var/lib/mugit/config.yaml
  meta:
    host: git.example.com
  repo:
    dir: /var/lib/mugit
  ```

  4. Systemd service
  ```bash
  cp mugit/mugit.service /etc/systemd/system/mugit.service
  systemctl enable --now mugit
  ```

  5. SSH server integration

  mugit integrates with the system's OpenSSH via `AuthorizedKeysCommand`. Add this to `/etc/ssh/sshd_config`:
  ```
  Match User mugit
    AuthorizedKeysCommand /usr/local/bin/mugit shell keys %f
    AuthorizedKeysCommandUser mugit
  ```

  Restart SSH:
  ```bash
  systemctl restart sshd
  ```

  5. Point reverse proxy to mugit, by default mugit runs on 8080 port.
</details>


## Configuration

mugit uses YAML for configuration. By default the server looks for a configuration file in this order (override with `-c` / `--config`):
1. `./config.yaml`
2. `/etc/mugit.yaml`
3. `/var/lib/mugit/config.yaml`


Durations follow Go's duration syntax (examples: `1h`, `30m`, `5s`). See: [https://pkg.go.dev/time#ParseDuration]

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
  log_file: /var/lib/mugit/mugit.log # where slog output is written (default: <repo.dir>/mugit.log)

meta:
  title: "My Git Server"    # site title shown on index page
  description: "A place for my projects"
  host: git.example.com     # used for clone URLs and go-import meta tag
  modt: "Welcome to my git server!" # message shown on SSH clone/push (empty = disabled)

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

# ssh: push/clone over SSH
ssh:
  enable: true
  user: "git" # user as which the app operates (default "git")
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
  # - from env: "$env:GITHUB_TOKEN" (will read $GITHUB_TOKEN)
  # - from file: "$file:/abs/path/to/token.txt"
  github_token: "$env:GITHUB_TOKEN"

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
mugit repo new myproject --description "My awesome project"

# toggle repository visibility
mugit repo private myproject

# show and set repository description
mugit repo description myproject
mugit repo description myproject "My awesome project"

# switch default branch
mugit repo set-default myproject main

# trigger mirror sync
mugit repo sync myproject
```

## License

mugit is licensed under the MIT License.
