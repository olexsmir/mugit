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
            vendorHash = "sha256-rY/O5padrE0cwwnvLIR3lM9xdpwloy0OFbp6/ge5gAc=";
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
          configFile =
            if cfg.configFile != null then cfg.configFile else format.generate "config.yaml" cfg.config;

          mugitWrapper = pkgs.symlinkJoin {
            name = "mugit";
            paths = [ cfg.package ];
            buildInputs = [ pkgs.makeWrapper ];
            postBuild = ''
              wrapProgram $out/bin/mugit \
                --add-flags "--config ${configFile}"
            '';
          };

          mugitWithCompletions = pkgs.stdenv.mkDerivation {
            pname = "mugit";
            version = cfg.package.version;
            src = mugitWrapper;
            nativeBuildInputs = [ pkgs.installShellFiles ];
            installPhase = ''
              mkdir -p $out/bin
              cp -r $src/bin/* $out/bin/

              installShellCompletion --cmd mugit \
                --bash <($out/bin/mugit completion bash) \
                --zsh <($out/bin/mugit completion zsh) \
                --fish <($out/bin/mugit completion fish)
            '';
          };
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

            exposeCli = mkOption {
              type = types.bool;
              default = true;
              description = "Whether to expose the mugit CLI to all users with the service configuration.";
            };

            dataDir = mkOption {
              type = types.path;
              default = "/var/lib/mugit";
              description = "Directory where mugit stores its data.";
            };

            configFile = mkOption {
              type = types.nullOr types.path;
              default = null;
              description = "Path to an existing mugit configuration file. Mutually exclusive with `config`.";
            };

            config = mkOption {
              type = format.type;
              default = { };
              description = ''
                Configuration for mugit. See documentation for available options.
                https://github.com/olexsmir/mugit/blob/main/README.md
              '';
              example = literalExpression ''
                {
                  server.port = 8080;
                  repo.dir = "/var/lib/mugit";
                }
              '';
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

          };

          config = mkIf cfg.enable {
            assertions = [
              {
                assertion = !(cfg.config != { } && cfg.configFile != null);
                message = "services.mugit: `config` and `configFile` are mutually exclusive. Only one can be set.";
              }
              {
                assertion = !(cfg.openFirewall && cfg.configFile != null);
                message = "services.mugit: `openFirewall` cannot be used with `configFile`. Set firewall rules manually or use `config` instead.";
              }
            ];

            environment.systemPackages = mkIf cfg.exposeCli [ mugitWithCompletions ];

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
              home = cfg.dataDir;
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
                  WorkingDirectory = cfg.dataDir;
                  StateDirectory = "mugit";
                  ExecStart = "${cfg.package}/bin/mugit serve --config ${configFile}";
                  Restart = "on-failure";
                  RestartSec = "5s";
                  NoNewPrivileges = true;
                  PrivateTmp = true;
                  ProtectSystem = "strict";
                  ProtectHome = true;
                  ReadWritePaths = [ cfg.dataDir ];
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
