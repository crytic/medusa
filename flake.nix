{
  description = "Medusa smart-contract fuzzer";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-26.05";
    utils.url = "github:numtide/flake-utils";
    crytic.url = "github:crytic/crytic.nix";
    crytic.inputs.nixpkgs.follows = "nixpkgs";
    crytic.inputs.utils.follows = "utils";
  };

  outputs = inputs: with inputs;
    utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs { inherit system; config.allowUnfree = true; };
      # Keep the compiler patched independently of the pinned nixpkgs snapshot.
      go126 = pkgs.go_1_26.overrideAttrs (_: {
        version = "1.26.10";
        src = pkgs.fetchurl {
          url = "https://go.dev/dl/go1.26.10.src.tar.gz";
          hash = "sha256-/WPog7ovK/ypPBZGsx6+9j4rHk0rMXyXidl2po2B828=";
        };
      });
    in rec {

      packages = {
        medusa = (pkgs.buildGo126Module.override { go = go126; }) {
          pname = "medusa";
          version = "1.5.1";
          src = ./.;
          vendorHash = "sha256-5RYxEN/0fd9z2nt3nqLzyG8pk7Q7Sz3fHka1toF/SFg=";
          nativeBuildInputs = [
            crytic.packages.${system}.crytic-compile
            crytic.packages.${system}.slither
            pkgs.solc
            pkgs.nodejs
          ];
          doCheck = false; # tests require `npm install` which can't run in hermetic build env
        };
        default = packages.medusa;
      };

      apps = {
        default = {
          type = "app";
          program = "${self.packages.${system}.medusa}/bin/medusa";
        };
      };

      devShells = {
        default = pkgs.mkShell {
          buildInputs = with pkgs; [
            packages.medusa
            bashInteractive
            # runtime dependencies
            crytic.packages.${system}.crytic-compile
            crytic.packages.${system}.slither
            solc
            # test dependencies
            nodejs
            # go development
            go126
            gotools
            go-tools
            gopls
            go-outline
            gopkgs
            gocode-gomod
            godef
            golint
            golangci-lint
            # formatting and linting
            dprint
            actionlint
          ];
        };
      };

    }
  );
}
