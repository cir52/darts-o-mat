# NixOS module: the server darts-o-mat as a service, like darts-o-mat.service of the Linux
# packages (tools/linuxpkg).
self:
{ config, lib, pkgs, ... }:

let
  cfg = config.services.darts-o-mat;
in
{
  options.services.darts-o-mat = {
    enable = lib.mkEnableOption "darts-o-mat, the darts scorer";

    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.stdenv.hostPlatform.system}.darts-o-mat;
      defaultText = lib.literalExpression "darts-o-mat.packages.\${system}.darts-o-mat";
      description = "The darts-o-mat package.";
    };

    port = lib.mkOption {
      type = lib.types.port;
      default = 3000;
      description = "Port of the web interface.";
    };

    openFirewall = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Open the port in the firewall.";
    };

    environmentFile = lib.mkOption {
      type = lib.types.nullOr lib.types.path;
      default = null;
      example = "/run/secrets/darts-o-mat";
      description = ''
        File with one NAME=value per line for the keys (DARTS_O_MAT_DSN,
        DARTS_O_MAT_TURNSTILE_SITE_KEY, ...), so they stay out of the Nix store and the
        process list. A DARTS_O_MAT_DSN here replaces the SQLite file in /var/lib/darts-o-mat.
      '';
    };

    extraFlags = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ ];
      example = [ "-db" "postgres" "-max-login-failures" "5" ];
      description = "More flags for darts-o-mat (see darts-o-mat -h).";
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.services.darts-o-mat = {
      description = "darts-o-mat: darts scorer";
      wants = [ "network-online.target" ];
      after = [ "network-online.target" ];
      wantedBy = [ "multi-user.target" ];
      # The games, in /var/lib/darts-o-mat. As a variable rather than -dsn, so that the
      # environment file can replace it (systemd lets it override Environment).
      environment.DARTS_O_MAT_DSN = "file:/var/lib/darts-o-mat/darts-o-mat.db";
      serviceConfig = {
        ExecStart = lib.escapeShellArgs ([ (lib.getExe cfg.package) "-addr" ":${toString cfg.port}" ] ++ cfg.extraFlags);
        EnvironmentFile = lib.mkIf (cfg.environmentFile != null) cfg.environmentFile;
        DynamicUser = true;
        StateDirectory = "darts-o-mat";
        Restart = "on-failure";
      };
    };

    networking.firewall.allowedTCPPorts = lib.mkIf cfg.openFirewall [ cfg.port ];
  };
}
