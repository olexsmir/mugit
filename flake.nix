{
  description = "a git server that your cow will love";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        version = self.rev or "dev";
      in
      {
        packages = {
          default = self.packages.${system}.mugit;
          mugit = pkgs.buildGoModule {
            pname = "mugit";
            version = version;
            src = ./.;
            vendorHash = "sha256-xF8IRS0Ne1zp4u6uolKFpKEZObSM6VhV95JUj2krXPY=";
            ldflags = [
              "-s"
              "-w"
              "-X main.version=${version}"
            ];
            meta = with pkgs.lib; {
              homepage = "https://github.com/olexsmir/mugit";
              license = licenses.mit;
            };
          };
        };
      }
    )
    // {
      nixosModules.default =
        {
          config,
          lib,
          pkgs,
          ...
        }:
        with lib;
        let
          cfg = config.services.mugit;
          format = pkgs.formats.yaml { };
          configFile = format.generate "config.yaml" cfg.config;
        in
        {
          options.services.mugit = {
            enable = mkEnableOption "mugit service";

            package = mkOption {
              type = types.package;
              default = self.packages.${pkgs.system}.mugit;
              defaultText = literalExpression "self.packages.\${pkgs.system}.mugit";
              description = "The mugit package to use.";
            };

            openFirewall = mkOption {
              type = types.bool;
              default = false;
              description = "Whether to open the firewall for mugit. Can only be used with `config`, not `configFile`.";
            };

            configFile = mkOption {
              type = types.nullOr types.path;
              default = null;
              description = "Path to an existing mugit configuration file. Mutually exclusive with `config`.";
            };

            user = mkOption {
              type = types.str;
              default = "mugit";
              description = "User account under which mugit runs.";
            };

            group = mkOption {
              type = types.str;
              default = "mugit";
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
                };
                options.repo = {
                  dir = mkOption {
                    type = types.str;
                    default = "";
                    description = "Directory which mugit will scan for repositories (required)";
                  };
                  masters = mkOption {
                    type = types.listOf types.str;
                    default = ["master" "main"];
                    description = "Master branch to look for";
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
                    description = "Wharever to run ssh server";
                  };
                  port = mkOption {
                    type = types.port;
                    default = 2222;
                    description = "Website port";
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
            networking.firewall = mkIf cfg.openFirewall {
              allowedTCPPorts =
                let
                  serverPort = cfg.config.server.port or 8080;
                  sshPort = cfg.config.ssh.port or 2222;
                  sshEnabled = cfg.config.ssh.enable or false;
                in
                [ serverPort ] ++ lib.optional sshEnabled sshPort;
            };

            users.users.${cfg.user} = {
              isSystemUser = true;
              group = cfg.group;
              home = cfg.config.repo.dir;
              createHome = true;
              description = "mugit service user";
            };

            users.groups.${cfg.group} = { };

            systemd.services.mugit = {
              description = "mugit service";
              wantedBy = [ "multi-user.target" ];
              after = [ "network.target" ];
              path = [ pkgs.git ];

              serviceConfig =
                let
                  serverPort = cfg.config.server.port or 8080;
                  sshPort = cfg.config.ssh.port or 2222;
                  sshEnabled = cfg.config.ssh.enable or false;
                  needsPrivPort = serverPort < 1024 || (sshEnabled && sshPort < 1024);
                in
                {
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
                }
                // lib.optionalAttrs needsPrivPort {
                  AmbientCapabilities = [ "CAP_NET_BIND_SERVICE" ];
                };
            };
          };
        };
    };
}
