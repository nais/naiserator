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

      naiseratorPackage = pkgs:
        pkgs.buildGoModule {
          pname = "naiserator";
          version = "0.0.0";
          src = pkgs.lib.sources.cleanSource ./.;
          # Hash is a product of the contents of all dependencies, aka Go modules
          vendorHash = "sha256-O33xQfgpVjYOCBF65Hi9W04NbdUE1vkGLfaD3WG98fI=";

          meta = with pkgs.lib; {
            description =
              "Naiserator creates a full set of Kubernetes application infrastructure based on a single application spec.";
            homepage = "https://github.com/nais/naiserator";
            license = licenses.mit;
            maintainers = [ "kimtore" ];
          };
        };

      # Lambda helper to provide system-specific attributes
      forAllSystems = f:
        nixpkgs.lib.genAttrs allSystems (system:
          f {
            pkgs = import nixpkgs {
              inherit system;
              # crossSystem = { config = "aarch64-unknown-linux-gnu"; };
            };
          });

      naiseratorLinuxPackage = pkgs:
        (pkgs.buildGoModule {
          pname = "naiserator";
          version = "0.0.0";
          src = pkgs.lib.sources.cleanSource ./.;
          # Hash is a product of the contents of all dependencies, aka Go modules
          vendorHash = "sha256-O33xQfgpVjYOCBF65Hi9W04NbdUE1vkGLfaD3WG98fI=";
          CGO_ENABLED = 0;
          doCheck = false;
          meta = with pkgs.lib; {
            description =
              "Naiserator creates a full set of Kubernetes application infrastructure based on a single application spec.";
            homepage = "https://github.com/nais/naiserator";
            license = licenses.mit;
            maintainers = [ "kimtore" ];
          };
        }).overrideAttrs (old:
          old // {
            GOOS = "linux";
            GOARCH = "amd64";
          });

    in {
      packages = forAllSystems ({ pkgs }: {
        # Build binaries for your system
        default = naiseratorLinuxPackage pkgs;

        # Build a docker image for your system
        dockerDarwin = pkgs.dockerTools.buildImage {
          name = "naiserator";
          created = "now";
          copyToRoot = with pkgs.dockerTools; [ ];

          config = {
            Cmd =
              [ "${naiseratorLinuxPackage pkgs}/bin/linux_amd64/naiserator" ];
          };
        };

        docker = pkgs.dockerTools.buildImage {
          name = "naiserator";
          created = "now";
          copyToRoot = with pkgs.dockerTools; [
            usrBinEnv
            binSh
            caCertificates
            fakeNss
          ];

          config = { Cmd = [ "${naiseratorPackage pkgs}/bin/naiserator" ]; };
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
