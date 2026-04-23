{
  description = "a git server that your cow will love";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs";
  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in {
      packages = forAllSystems (pkgs:
        let version = self.rev or "dev";
        in {
          default = self.packages.${pkgs.stdenv.hostPlatform.system}.mugit;
          mugit = pkgs.buildGo126Module {
            pname = "mugit";
            version = version;
            src = ./.;
            vendorHash = "sha256-qVvVlS0RgOms6/o6TfGxdQ1DYhr1ouWY7UhAWYN/6ps=";
            ldflags = [ "-s" "-w" "-X main.version=${version}" ];
            nativeCheckInputs = [ pkgs.git ];
            meta = with pkgs.lib; {
              homepage = "https://git.olexsmir.xyz/mugit";
              license = licenses.mit;
            };
          };
        }
      );

      nixosModules.default = { config, lib, pkgs, ... }:
        with lib;
        let
          cfg = config.services.mugit;
          format = pkgs.formats.yaml { };
          sshUser = if cfg.config.ssh.user != "" then cfg.config.ssh.user else cfg.user;
          configFile  = if cfg.configFile != null
            then cfg.configFile
            else format.generate "config.yaml" cfg.config;
        in {
          options.services.mugit = {
            enable = mkEnableOption "mugit service";

            package = mkOption {
              type = types.package;
              default = self.packages.${pkgs.stdenv.hostPlatform.system}.mugit;
              defaultText = literalExpression "self.packages.\${pkgs.stdenv.hostPlatform.system}.mugit";
              description = "The mugit package to use.";
            };

            exposeCli = mkOption {
              type = types.bool;
              default = false;
              description = "Whether to expose a mugit CLI wrapper to all system users, runs as the mugit user/group.";
            };

            configFile = mkOption {
              type = types.nullOr types.path;
              default = null;
              description = "Path to an existing mugit configuration file. Mutually exclusive with `config`.";
            };

            user = mkOption {
              type = types.str;
              default = "git";
              description = "User account under which mugit runs.";
            };

            group = mkOption {
              type = types.str;
              default = "git";
              description = "Group under which mugit runs.";
            };

            config = mkOption {
              default = {};
              description = ''
                The primary mugit configuration.
                See [docs](https://github.com/olexsmir/mugit) for possible values.
              '';
              example = literalExpression ''
                {
                  meta.host = "git.example.org";
                  repo.dir = "/var/lib/mugit";
                  ssh = {
                    enable = true;
                    host_key = "/var/lib/mugit/key";
                  };
                }
              '';
              type = types.submodule {
                options.meta = {
                  title = mkOption {
                    type = types.str;
                    default = "mugit";
                    description = "Website title";
                  };
                  description = mkOption {
                    type = types.str;
                    default = "";
                    description = "Website description";
                  };
                  host = mkOption {
                    type = types.str;
                    default = "";
                    description = "Website CNAME (required)";
                  };
                };
                options.server = {
                  host = mkOption {
                    type = types.str;
                    default = "";
                    description = "Host address";
                  };
                  port = mkOption {
                    type = types.port;
                    default = 8080;
                    description = "Website port";
                  };
                  log_file = mkOption {
                    type = types.str;
                    default = "";
                    description = "File to write mugit logs";
                  };
                };
                options.repo = {
                  dir = mkOption {
                    type = types.str;
                    default = "";
                    description = "Directory which mugit will scan for repositories (required)";
                  };
                  readmes = mkOption {
                    type = types.listOf types.str;
                    default = ["README.md" "readme.md" "README.html" "readme.html" "README.txt" "readme.txt" "readme"];
                    description = "Readme files to look for";
                  };
                };
                options.ssh = {
                  enable = mkOption {
                    type = types.bool;
                    default = false;
                    description = "Whether to enable SSH git access";
                  };
                  user = mkOption {
                    type = types.str;
                    default = "";
                    description = "User used for git access. Defaults to the main user option.";
                  };
                  host_key = mkOption {
                    type = types.str;
                    default = "";
                    description = "Path to ssh private key (required if ssh enabled)";
                  };
                  keys = mkOption {
                    type = types.listOf types.str;
                    default = [];
                    description = "List of public ssh keys which are allows to do git pushes, and access private repositories";
                  };
                };
                options.mirror = {
                  enable = mkOption {
                    type = types.bool;
                    default = false;
                    description = "Wharever to run mirroring worker";
                  };
                  interval = mkOption {
                    type = types.str;
                    default = "8h";
                    description = "Interval in which mirroring will happen";
                  };
                  github_token = mkOption {
                    type = types.str;
                    default = "";
                    description = "Github token for pulling from github repos";
                  };
                };
                options.cache = {
                  home_page = mkOption {
                    type = types.str;
                    default = "5m";
                    description = "For how long index page is cached";
                  };
                  readme = mkOption {
                    type = types.str;
                    default = "1m";
                    description = "For how long repos readme is cached";
                  };
                };
              };
            };
          };

          config = mkIf cfg.enable {
            assertions = [
              {
                assertion = !cfg.config.ssh.enable || (cfg.config.ssh.keys != []);
                message = "SSH is enabled but no SSH keys provided. Please add keys to services.mugit.config.ssh.keys";
              }
            ];

            users.groups.${cfg.group} = { };
            users.users.${cfg.user} = {
              isSystemUser = true;
              useDefaultShell = true;
              group = cfg.group;
              home = cfg.config.repo.dir;
              createHome = true;
              description = "mugit service user";
            };

            services.openssh = mkIf cfg.config.ssh.enable {
              enable = true;
              extraConfig = ''
                Match User ${sshUser}
                    AuthorizedKeysCommand /etc/ssh/mugit_authorized_keys
                    AuthorizedKeysCommandUser ${sshUser}
                    ChallengeResponseAuthentication no
                    PasswordAuthentication no
                    AllowUsers ${sshUser}
              '';
            };

            environment.etc."ssh/mugit_authorized_keys" = mkIf cfg.config.ssh.enable {
              mode = "0555";
              text = ''
                #!${pkgs.stdenv.shell}
                ${cfg.package}/bin/mugit --config ${configFile} shell keys "$1"
              '';
            };

            environment.systemPackages = lib.mkIf cfg.exposeCli [
              (pkgs.runCommandLocal "mugit-completions" {} ''
                mkdir -p $out/share/bash-completion/completions
                mkdir -p $out/share/zsh/site-functions
                mkdir -p $out/share/fish/vendor_completions.d
                ${cfg.package}/bin/mugit completion bash > $out/share/bash-completion/completions/mugit
                ${cfg.package}/bin/mugit completion zsh  > $out/share/zsh/site-functions/_mugit
                ${cfg.package}/bin/mugit completion fish > $out/share/fish/vendor_completions.d/mugit.fish
              '')
            ];

            security.wrappers = lib.mkIf cfg.exposeCli {
              mugit = {
                source =
                  let
                    mugitWrapped = pkgs.writeScriptBin "mugit" ''
                      #!${pkgs.bash}/bin/bash
                      exec ${cfg.package}/bin/mugit --config ${configFile} "$@"
                    '';
                  in
                  "${mugitWrapped}/bin/mugit";
                owner = cfg.user;
                group = cfg.group;
                setuid = true;
                setgid = true;
                permissions = "u+rx,g+rx,o+rx";
              };
            };

            systemd.services.mugit = {
              description = "mugit service";
              wantedBy = [ "multi-user.target" ];
              after = [ "network.target" ] ++ lib.optionals cfg.config.ssh.enable [ "sshd.service" ];
              path = [ pkgs.git ];
              serviceConfig = {
                Type = "simple";
                User = cfg.user;
                Group = cfg.group;
                WorkingDirectory = cfg.config.repo.dir;
                StateDirectory = "mugit";
                ExecStart = "${cfg.package}/bin/mugit serve --config ${configFile}";
                Restart = "on-failure";
                RestartSec = "5s";
                NoNewPrivileges = true;
                PrivateTmp = true;
                ProtectSystem = "strict";
                ProtectHome = true;
                ReadWritePaths = [ cfg.config.repo.dir ];
                ProtectKernelTunables = true;
                ProtectKernelModules = true;
                ProtectControlGroups = true;
              };
            };
          };
        };
    };
}
