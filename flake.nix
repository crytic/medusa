{
  description = "Medusa smart-contract fuzzer";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-25.05";
    utils.url = "github:numtide/flake-utils";
    crytic.url = "github:crytic/crytic.nix";
    crytic.inputs.nixpkgs.follows = "nixpkgs";
    crytic.inputs.utils.follows = "utils";
  };

  outputs = inputs: with inputs;
    utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs { inherit system; config.allowUnfree = true; };
    in rec {

      packages = {
        medusa = pkgs.buildGoModule {
          pname = "medusa";
          version = "1.4.1";
          src = ./.;
          vendorHash = "sha256-JaYAZX/u8cArVnQChmotpTbxja6XY0OaPr/RPd7rVXc=";
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
            go
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
