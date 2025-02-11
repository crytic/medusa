# Installation

There are three main ways to install `medusa` at the moment. The first is using Homebrew,
building from source, or installing a precompiled binary.

If you have any difficulty with installing `medusa`, please [open an issue](https://github.com/crytic/medusa/issues) on GitHub.

## Installing with Homebrew

Note that using Homebrew is only viable (and recommended) for macOS and Linux users. For Windows users, you must
[build from source](#building-from-source) or [install a precompiled binary](#precompiled-binaries).

### Prerequisites

Installation instructions for Homebrew can be found [here](https://brew.sh/).

### Install `medusa`

Run the following command to install `medusa`:

```shell
brew install medusa
```

## Installing with Nix

### Prerequisites

Make sure nix is installed and that `nix-command` and `flake` features are enabled. The [Determinate Systems nix-installer](https://determinate.systems/nix-installer/) will automatically enable these features and is the recommended approach. If nix is already installed without these features enabled, run the following commands.

```
mkdir -p ~/.config/nix
echo 'experimental-features = nix-command flakes' > ~/.config/nix/nix.conf
```

### Build `medusa`

`nix build` will build medusa and wire up independent copies of required dependencies. The resulting binary can be found at `./result/bin/medusa`

### Install `medusa`

After building, you can add the build result to your PATH using nix profiles by running the following command:

`nix profile install ./result`

## Building from source

### Prerequisites

Before downloading `medusa`, you will need to download Golang and `crytic-compile`.

- Installation instructions for Golang can be found [here](https://go.dev/doc/install)
- Installation instructions for `crytic-compile` can be found [here](https://github.com/crytic/crytic-compile#installation)
  - Note that `crytic-compile` requires a Python environment. Installation instructions for Python can be found
    [here](https://www.python.org/downloads/).

### Build `medusa`

Run the following commands to build `medusa` (this should work on all OSes):

```shell
# Clone the repository
git clone https://github.com/crytic/medusa

# Build medusa
cd medusa
go build -trimpath
```

You will now need to move the binary (`medusa` or `medusa.exe`) to somewhere in your `PATH` environment variable so that
it is accessible via the command line. Please review the instructions
[here](https://zwbetz.com/how-to-add-a-binary-to-your-path-on-macos-linux-windows/) (if you are a Windows user, we
recommend using the Windows GUI).

## Precompiled binaries

The precompiled binaries can be downloaded on `medusa`'s [GitHub releases page](https://github.com/crytic/medusa/releases).

> **_NOTE:_** macOS may set the [quarantine extended attribute](https://superuser.com/questions/28384/what-should-i-do-about-com-apple-quarantine)
> on the downloaded zip file. To remove this attribute, run the following command:
> `sudo xattr -rd com.apple.quarantine <my_file.tar.gz>`.

Once installed, you will need to unzip the file and move the binary to somewhere in your `$PATH`. Please review the instructions
[here](https://zwbetz.com/how-to-add-a-binary-to-your-path-on-macos-linux-windows/) (if you are a Windows user, we
recommend using the Windows GUI).
