{ lib
, buildGoModule
, go_1_27
, makeWrapper
, src
, vendorHash
, version
, glib
, gtk4
, webkitgtk_6_0
, libjpeg_turbo
}:

(buildGoModule.override { go = go_1_27; }) {
  pname = "darts-o-cam";
  inherit src vendorHash version;

  subPackages = [ "cmd/darts-o-cam" ];
  # purego loads WebKitGTK and libjpeg-turbo at runtime, no cgo.
  env.CGO_ENABLED = 0;
  ldflags = [
    "-s"
    "-w"
    "-X github.com/sj14/darts-o-mat/internal/common/buildinfo.stamped=${version}"
  ];

  nativeBuildInputs = [ makeWrapper ];

  postInstall = ''
    install -Dm644 tools/linuxpkg/darts-o-cam.desktop $out/share/applications/darts-o-cam.desktop
    install -Dm644 internal/common/static/favicon.svg $out/share/icons/hicolor/scalable/apps/darts-o-cam.svg
  '';

  # NixOS has no library search path: the window (glaze) loads WebKitGTK and the cameras
  # libjpeg-turbo with dlopen. Without them darts-o-cam opens the browser, or decodes slower.
  postFixup = ''
    wrapProgram $out/bin/darts-o-cam \
      --prefix LD_LIBRARY_PATH : ${lib.makeLibraryPath [ glib gtk4 webkitgtk_6_0 libjpeg_turbo ]}
  '';

  meta = {
    description = "Detects darts with cameras at the board and enters them in darts-o-mat";
    homepage = "https://github.com/sj14/darts-o-mat";
    license = lib.licenses.agpl3Only;
    mainProgram = "darts-o-cam";
    platforms = lib.platforms.linux;
  };
}
