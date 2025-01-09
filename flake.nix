{
  description = "Medusa smart-contract fuzzer";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-24.11";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; config.allowUnfree = true; };
        pyCommon = {
          format = "pyproject";
          nativeBuildInputs = with pkgs.python3Packages; [ pythonRelaxDepsHook ];
          pythonRelaxDeps = true;
          doCheck = false;
        };
      in
      rec {

        packages = rec {

          solc-select = pkgs.python3Packages.buildPythonPackage (pyCommon // {
            pname = "solc-select";
            version = "1.0.4";
            src = builtins.fetchGit {
              url = "https://github.com/crytic/solc-select.git";
              rev = "8072a3394bdc960c0f652fb72e928a7eae3631da";
            };
            propagatedBuildInputs = with pkgs.python3Packages; [
              packaging
              setuptools
              pycryptodome
            ];
          });

          crytic-compile = pkgs.python3Packages.buildPythonPackage (pyCommon // rec {
            pname = "crytic-compile";
            version = "0.3.7";
            src = builtins.fetchGit {
              url = "https://github.com/crytic/crytic-compile.git";
              rev = "20df04f37af723eaa7fa56dc2c80169776f3bc4d";
            };
            propagatedBuildInputs = with pkgs.python3Packages; [
              cbor2
              pycryptodome
              setuptools
              packages.solc-select
            ];
          });

          slither = pkgs.python3Packages.buildPythonPackage (pyCommon // rec {
            pname = "slither";
            version = "0.10.4";
            format = "pyproject";
            src = builtins.fetchGit {
              url = "https://github.com/crytic/slither.git";
              rev = "aeeb2d368802844733671e35200b30b5f5bdcf5c";
            };
            nativeBuildInputs = with pkgs.python3Packages; [ pythonRelaxDepsHook ];
            pythonRelaxDeps = true;
            doCheck = false;
            propagatedBuildInputs = with pkgs.python3Packages; [
              packaging
              prettytable
              pycryptodome
              packages.crytic-compile
              web3
            ];
          });

          medusa = pkgs.buildGoModule {
            pname = "medusa";
            version = "0.1.8"; # from cmd/root.go
            src = ./.;
            vendorHash = "sha256-12Xkg5dzA83HQ2gMngXoLgu1c9KGSL6ly5Qz/o8U++8=";
            nativeBuildInputs = [
              packages.crytic-compile
              pkgs.solc
              pkgs.nodejs
            ];
            doCheck = false; # tests require `npm install` which can't run in hermetic build env
          };

          default = medusa;

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
              packages.crytic-compile
              packages.slither
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
            ];
          };
        };

      }
    );
}
