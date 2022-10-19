{
  description = "dwm - dynamic window manager";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-unstable";
  };

  outputs = inputs@{ nixpkgs, ... }:
  let
    system = "x86_64-linux";

    pkgs = import nixpkgs { inherit system; };

    lib = nixpkgs.lib;

    go = [
      pkgs.go
    ];

    tools = [
      pkgs.gopkgs
      pkgs.gopls
      pkgs.golint
      pkgs.go-outline
      pkgs.gotools
      pkgs.entr
    ];

    site = pkgs.buildGoModule {
      name = "site";
      src = ./.;
      vendorSha256 = "sha256-xPsCwRHeh9HdrR6LwD2kz1w+SslfRgaHqV9MBwlDnNs=";
      runVend = true;

      nativeBuildInputs = [ pkgs.musl ];
      CGO_ENABLED = 0;
      ldflags = [
        "-linkmode external"
        "-extldflags '-static -L${pkgs.musl}/lib'"
      ];
    };


  in rec {

    defaultPackage.x86_64-linux = pkgs.stdenv.mkDerivation {
      name = "kaixi-site";
      src = ./.;
      propagatedBuildInputs = [ site ];
      installPhase = ''
        mkdir -p "$out/bin/"
        mkdir -p "$out/share/site/"
        mv config.toml markdown static templates $out/share/site/
        printf "#!/bin/sh\ncd $out/share/site\n${site}/bin/site\n" > $out/bin/site.sh
        chmod +x $out/bin/site.sh
      '';
    };

    docker = pkgs.dockerTools.buildImage {
      name = "site";
      copyToRoot = [ pkgs.bash pkgs.toybox defaultPackage.x86_64-linux ];
      config = {
        Cmd = [ "${defaultPackage.x86_64-linux}/bin/site.sh" ];
        ExposedPorts = { "8080/tcp" = {}; };
      };
    };

    devShells.x86_64-linux.default = pkgs.mkShell {
      buildInputs = go ++ tools;
    };
  };
}
