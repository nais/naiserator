{
  description = "Naiserator";

  # Flake inputs
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs"; # also valid: "nixpkgs"
  };

  # Flake outputs
  outputs = { self, nixpkgs }:
    let
      # Systems supported
      allSystems = [
        "x86_64-linux" # 64-bit Intel/AMD Linux
        "aarch64-linux" # 64-bit ARM Linux
        "x86_64-darwin" # 64-bit Intel macOS
        "aarch64-darwin" # 64-bit ARM macOS
      ];

      # Lambda helper to provide system-specific attributes
      forAllSystems = f: nixpkgs.lib.genAttrs allSystems (system: f {
        pkgs = import nixpkgs { inherit system; };
      });

    # Development environment output
    in
    {
      packages = forAllSystems ({ pkgs }: {
        default = pkgs.buildGoModule {
          pname = "naiserator";
          version = "0.0.0";
          src = ./.;
          # Hash is a product of the contents of all dependencies, aka Go modules
          vendorHash = "sha256-OkSvMBo+TSRTIaPF6+wCOQ7YiaAoa+eU2ft5Z4E7Fpw=";

          meta = with pkgs.lib; {
            description = "Naiserator creates a full set of Kubernetes application infrastructure based on a single application spec.";
            homepage = "https://github.com/nais/naiserator";
            license = licenses.mit;
            maintainers = ["kimtore"];
          };
        };
      });
      devShells = forAllSystems ({ pkgs }: {
        default = pkgs.mkShell {
          # The Nix packages provided in the environment
          packages = with pkgs; [
            go_1_22 # Go 1.22
            gotools # Go tools like goimports, godoc, and others
          ];
        };
      });
    };
}
