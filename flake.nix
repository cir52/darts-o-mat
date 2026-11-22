{
  description = "darts-o-mat, a self-hosted darts scorer, and darts-o-cam, which detects the darts with cameras at the board";

  # nixos-unstable: go.mod needs Go 1.27.
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      lib = nixpkgs.lib;
      # Linux only: the camera's window needs WebKitGTK and its cameras V4L2.
      systems = [ "x86_64-linux" "aarch64-linux" ];
      forAllSystems = f: lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});

      # What both programs are built from. Only what the build needs, so that a change of
      # the project page or the docs doesn't build them again.
      common = {
        src = lib.fileset.toSource {
          root = ./.;
          fileset = lib.fileset.unions [
            ./go.mod
            ./go.sum
            ./cmd
            ./internal
            ./tools/linuxpkg/darts-o-cam.desktop
          ];
        };
        # The module's dependencies, the same for both programs: update it whenever go.sum
        # changes (the failing build tells the new one).
        vendorHash = "sha256-pOwF+YYCgMCQxbcw2JoGd4Rr69MH5KaQWF66sP6vhA0=";
        # A revision instead of a release number: then the programs don't show the hint
        # about a newer release, Nix updates them.
        version = self.shortRev or self.dirtyShortRev or "dev";
      };
    in
    {
      packages = forAllSystems (pkgs: rec {
        darts-o-mat = pkgs.callPackage ./nix/darts-o-mat.nix common;
        darts-o-cam = pkgs.callPackage ./nix/darts-o-cam.nix common;
        # What most need: the server usually runs already somewhere.
        default = darts-o-cam;
      });

      nixosModules.darts-o-mat = import ./nix/darts-o-mat-module.nix self;
      nixosModules.darts-o-cam = import ./nix/darts-o-cam-module.nix self;
    };
}
