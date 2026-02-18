{
  description = "vop - AWS credential management via 1Password";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        version = "0.2.0";
      in
      {
        packages = {
          vop = pkgs.buildGoModule {
            pname = "vop";
            inherit version;
            src = ./.;

            # To update: run `nix build` and replace with the hash from the error message.
            vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";

            CGO_ENABLED = "1";

            tags = [ "noupdate" ];

            ldflags = [
              "-s" "-w"
              "-X github.com/NodeSpy/vop/internal/version.Version=v${version}"
              "-X github.com/NodeSpy/vop/internal/version.Date=1970-01-01T00:00:00Z"
            ];

            meta = with pkgs.lib; {
              description = "AWS credential management via 1Password";
              homepage = "https://github.com/NodeSpy/vop";
              license = licenses.mit;
              mainProgram = "vop";
            };
          };

          default = self.packages.${system}.vop;
        };

        # Development shell with Go and build tools
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gcc
          ];
        };
      }
    );
}
