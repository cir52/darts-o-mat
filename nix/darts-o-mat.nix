{ lib
, buildGoModule
, go_1_27
, src
, vendorHash
, version
}:

(buildGoModule.override { go = go_1_27; }) {
  pname = "darts-o-mat";
  inherit src vendorHash version;

  subPackages = [ "cmd/darts-o-mat" ];
  env.CGO_ENABLED = 0;
  ldflags = [
    "-s"
    "-w"
    "-X github.com/sj14/darts-o-mat/internal/common/buildinfo.stamped=${version}"
  ];

  meta = {
    description = "Self-hosted darts scorer: web interface, accounts, games and statistics";
    homepage = "https://github.com/sj14/darts-o-mat";
    license = lib.licenses.agpl3Only;
    mainProgram = "darts-o-mat";
    platforms = lib.platforms.linux;
  };
}
