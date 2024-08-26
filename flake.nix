{
  description = "Fasit";

  inputs.nixpkgs.url = "nixpkgs/nixos-unstable";

  outputs = {nixpkgs, ...}: let
    goOverlay = final: prev: {
      go = prev.go.overrideAttrs (old: {
        version = "1.23.0";
        src = prev.fetchurl {
          url = "https://go.dev/dl/go1.23.0.src.tar.gz";
          hash = "sha256-Qreo6A2AXaoDAi7T/eQyHUw78smQoUQWXQHu7Nb2mcY=";
        };
      });
    };
    withSystem = nixpkgs.lib.genAttrs ["x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin"];
    withPkgs = callback:
      withSystem (
        system:
          callback
          (import nixpkgs {
            inherit system;
            overlays = [goOverlay];
          })
      );
  in {
    devShells = withPkgs (pkgs: {
      default = pkgs.mkShell {
        buildInputs = with pkgs; [
          go
          gopls
          gotools
          go-tools
          gnumake
		  gofumpt
        ];
      };
    });

    formatter = withPkgs (pkgs: pkgs.nixfmt-rfc-style);
  };
}