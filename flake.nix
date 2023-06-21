{
  description = "Medusa smart-contract fuzzer";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-22.11";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; config.allowUnfree = true; };
        pyCommon = {
          format = "pyproject";
          nativeBuildInputs = with pkgs.python39Packages; [ pythonRelaxDepsHook ];
          pythonRelaxDeps = true;
          doCheck = false;
        };
      in
      rec {

        packages = rec {

          solc-select = pkgs.python39Packages.buildPythonPackage (pyCommon // {
            pname = "solc-select";
            version = "1.0.3";
            src = builtins.fetchGit {
              url = "git+ssh://git@github.com/crytic/solc-select";
              rev = "97f160611c39d46e27d6f44a5a61344e6218d584";
            };
            propagatedBuildInputs = with pkgs.python39Packages; [
              packaging
              setuptools
              pycryptodome
            ];
          });

          crytic-compile = pkgs.python39Packages.buildPythonPackage (pyCommon // rec {
            pname = "crytic-compile";
            version = "0.3.1";
            src = builtins.fetchGit {
              url = "git+ssh://git@github.com/crytic/crytic-compile";
              rev = "10104f33f593ab82ba5780a5fe8dd26385acd1c1";
            };
            propagatedBuildInputs = with pkgs.python39Packages; [
              cbor2
              pycryptodome
              setuptools
              packages.solc-select
            ];
          });

          slither = pkgs.python39Packages.buildPythonPackage (pyCommon // rec {
            pname = "slither";
            version = "0.9.3";
            format = "pyproject";
            src = builtins.fetchGit {
              url = "git+ssh://git@github.com/crytic/slither";
              rev = "e6b8af882c6419a9119bec5f4cfff93985a92f4e";
            };
            nativeBuildInputs = with pkgs.python39Packages; [ pythonRelaxDepsHook ];
            pythonRelaxDeps = true;
            doCheck = false;
            propagatedBuildInputs = with pkgs.python39Packages; [
              packaging
              prettytable
              pycryptodome
              packages.crytic-compile
            ];
            postPatch = ''
              echo "web3 dependency depends on ipfs which is bugged, removing it from the listed deps"
              sed -i 's/"web3>=6.0.0",//' setup.py
            '';
          });

          medusa = pkgs.buildGoModule {
            pname = "medusa";
            version = "0.1.0"; # from cmd/root.go
            src = ./.;
            vendorSha256 = "sha256-odBzty8wgFfdSF18D15jWtUNeQPJ7bkt9k5dx+8EFb4=";
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
              gocode
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
