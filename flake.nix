{
  description = "development workspace";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        devShells.default = pkgs.mkShell {
          # hardeningDisable = [ "all" ];

          buildInputs = with pkgs; [
            (stdenv.mkDerivation rec {
              name = "run";
              pname = "run";
              src = fetchurl {
                url = "https://github.com/nxtcoder17/Runfile/releases/download/v1.1.2/run-linux-amd64";
                sha256 = "sha256-SKLuX/i86cpryK1TFOrtXGLikikwEuH0dPU4xur6N/Y=";
              };
              unpackPhase = ":";
              installPhase = ''
                mkdir -p $out/bin
                cp $src $out/bin/$name
                chmod +x $out/bin/$name
              '';
            })

            # version control
            git

            # programming tools
            go
            kubebuilder
          ];

          shellHook = ''
            export PATH="$PWD/bin:$PATH"
          '';
        };
      }
    );
}
