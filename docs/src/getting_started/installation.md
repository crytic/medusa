# Installation

There are six main ways to install `medusa`:
1. [Using Go Install](#installing-with-go-install)
2. [Using Homebrew](#installing-with-homebrew)
3. [Using Nix](#installing-with-nix)
4. [Using Docker](#installing-with-docker)
5. [Building from source](#building-from-source)
6. [Using Precompiled binaries](#precompiled-binaries)

If you have any difficulty with installing `medusa`, please [open an issue](https://github.com/crytic/medusa/issues) on GitHub.

## Installing with Go Install

### Prerequisites

You need to have Go installed (version 1.20 or later). Installation instructions for Go can be found [here](https://go.dev/doc/install).

### Install `medusa`

Run the following command to install the latest version of `medusa`:

```shell
go install github.com/crytic/medusa@latest
```

This will download, compile, and install the `medusa` binary to your `$GOPATH/bin` directory. Make sure this directory is in your `PATH` environment variable.

## Installing with Homebrew

Note that using Homebrew is only viable (and recommended) for macOS and Linux users. For Windows users, you must
use one of the other installation methods.

### Prerequisites

Installation instructions for Homebrew can be found [here](https://brew.sh/).

### Install `medusa`

Run the following command to install the latest stable release of `medusa`:

```shell
brew install medusa
```

To install the latest development version:

```shell
brew install --HEAD medusa
```

## Installing with Nix

### Prerequisites

Make sure nix is installed and that `nix-command` and `flake` features are enabled. The [Determinate Systems nix-installer](https://determinate.systems/nix-installer/) will automatically enable these features and is the recommended approach. If nix is already installed without these features enabled, run the following commands:

```shell
mkdir -p ~/.config/nix
echo 'experimental-features = nix-command flakes' > ~/.config/nix/nix.conf
```

### Build `medusa`

To build medusa with nix:

```shell
# Clone the repository if you haven't already
git clone https://github.com/crytic/medusa
cd medusa

# Build medusa
nix build
```

The resulting binary can be found at `./result/bin/medusa`.

### Install `medusa`

After building, you can add the build result to your PATH using nix profiles by running the following command:

```shell
nix profile install ./result
```

## Installing with Docker

### Prerequisites

You need to have Docker installed. Installation instructions for Docker can be found [here](https://docs.docker.com/get-docker/).

### Using the Docker image

Pull the latest Docker image:

```shell
docker pull crytic/medusa
```

Run medusa in a container:

```shell
docker run -it --rm -v $(pwd):/src crytic/medusa <command>
```

This will mount your current directory to `/src` in the container and run the specified medusa command.

## Building from source

### Prerequisites

Before building `medusa` from source, you will need:

- Go (version 1.20 or later) - [Installation instructions](https://go.dev/doc/install)
- `crytic-compile` - [Installation instructions](https://github.com/crytic/crytic-compile#installation)
  - Note that `crytic-compile` requires a Python environment - [Python installation instructions](https://www.python.org/downloads/)
- `slither` (Optional) - For improved valuegeneration we recommend also [installing Slither](https://github.com/crytic/slither?tab=readme-ov-file#how-to-install)

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

Once downloaded, you will need to unzip the file and move the binary to somewhere in your `$PATH`. Please review the instructions
[here](https://zwbetz.com/how-to-add-a-binary-to-your-path-on-macos-linux-windows/) (if you are a Windows user, we
recommend using the Windows GUI).