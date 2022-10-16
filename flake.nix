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

  in rec {

    # Gotta clean this up
    defaultPackage.x86_64-linux = pkgs.stdenv.mkDerivation {
      name = "site";
      src = ./.;
      buildInputs = go;
      installPhase = ''
        export GOCACHE=$TMPDIR/go-cache
        export GOPATH="$TMPDIR/go"
        mkdir -p "$out/bin/"
        mkdir -p "$out/share/site/"
        mv static templates $out/share/site/
        go build
        mv site $out/bin/
        printf "#!/bin/sh\ncd $out/share/site\n$out/bin/site\n" > $out/bin/site.sh
        chmod +x $out/bin/site.sh
      '';
    };

    docker = pkgs.dockerTools.buildImage {
      name = "site";
      copyToRoot = [ pkgs.bash pkgs.toybox ];
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
