# NixOS module: darts-o-cam as a service without a desktop, like darts-o-cam.service of the
# Linux packages (tools/linuxpkg). On a desktop the package alone is enough
# (environment.systemPackages), with the user in the group video.
self:
{ config, lib, pkgs, ... }:

let
  cfg = config.services.darts-o-cam;
in
{
  options.services.darts-o-cam = {
    enable = lib.mkEnableOption "darts-o-cam, detecting darts with the board's cameras";

    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.stdenv.hostPlatform.system}.darts-o-cam;
      defaultText = lib.literalExpression "darts-o-mat.packages.\${system}.darts-o-cam";
      description = "The darts-o-cam package.";
    };

    port = lib.mkOption {
      type = lib.types.port;
      default = 8090;
      description = "Port of the web interface, for the calibration from other devices.";
    };

    openFirewall = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Open the port in the firewall. The web interface has no login: only in a network you trust.";
    };

    extraFlags = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ ];
      example = [ "-v" ];
      description = "More flags for darts-o-cam (see darts-o-cam -h).";
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.services.darts-o-cam = {
      description = "darts-o-cam: detects darts with the board's cameras";
      wants = [ "network-online.target" ];
      after = [ "network-online.target" ];
      wantedBy = [ "multi-user.target" ];
      serviceConfig = {
        # Configuration and calibration in /var/lib/darts-o-cam.
        ExecStart = lib.escapeShellArgs ([ (lib.getExe cfg.package) "-ui" "none" "-addr" ":${toString cfg.port}" "-dir" "/var/lib/darts-o-cam" ] ++ cfg.extraFlags);
        DynamicUser = true;
        # The cameras.
        SupplementaryGroups = [ "video" ];
        StateDirectory = "darts-o-cam";
        Restart = "on-failure";
      };
    };

    networking.firewall.allowedTCPPorts = lib.mkIf cfg.openFirewall [ cfg.port ];
  };
}
