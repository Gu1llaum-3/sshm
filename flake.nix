{
  description = "SSHM - A modern, interactive SSH Manager for your terminal";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        version = "1.10.0";
      in
      {
        packages = {
          sshm = pkgs.buildGoModule {
            pname = "sshm";
            version = version;

            src = ./.;

            vendorHash = "sha256-aU/+bxcETs/Jq5FVAdiioyuc1AufvWeiqFQ7uo1cK1k=";

            doCheck = false;

            ldflags = [
              "-s"
              "-w"
              "-X main.version=${version}"
            ];

            nativeBuildInputs = [ pkgs.go ];

            meta = with pkgs.lib; {
              description = "A modern, interactive SSH Manager for your terminal";
              homepage = "https://github.com/Gu1llaum-3/sshm";
              license = licenses.mit;
              platforms = platforms.unix ++ platforms.windows;
            };
          };

          default = self.packages.${system}.sshm;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = [ pkgs.go ];
        };

        apps.default = {
          type = "app";
          program = "${self.packages.${system}.sshm}/bin/sshm";
        };
      }
    );
}
