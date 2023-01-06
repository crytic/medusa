# This Dockerfile is used to create a Docker container image for the Medusa tool,
# which is developed by Trail of Bits. The image is based on the Ubuntu Jammy
# Jellyfish distribution and has various dependencies installed, including Node.js,
# solc-install, Solidity compiler (solc), Truffle, Hardhat, and Go.

# Multi-stage building

# A stage to get Golang dependencies
FROM medusa AS go_get
RUN go get ./...

# A stage to run unit_tests and export them to the medusa_unit_tests.log file
FROM go_get AS unit_tests_stage
RUN go test -v ./... > medusa_unit_tests.log

# A stage to save only unit tests log
FROM scratch AS unit_tests_log
COPY --from=unit_tests_stage /medusa/medusa_unit_tests.log .

# A stage to have medusa built
FROM go_get AS medusa_build
RUN go build

# Use the Ubuntu Jammy Jellyfish distribution as the base image
# and assign the name "medusa" to it
FROM ubuntu:jammy as medusa

# Set metadata about the image and its maintainer
LABEL "maintainer"="Trail of Bits"
LABEL "about"="medusa container image"

# Define build arguments that can be passed to the Docker build command
# (using the `--build-arg` argument). The default values for these arguments
# will be used if no values are specified
ARG NODE_VERSION=18.13.0
ARG SOLC_VERSION=0.8.17
ARG GO_VERSION=1.19.4
ARG HARDHAT_VERSION=latest
ARG TRUFFLE_VERSION=latest

# Update the package list and install necessary ones
# `build-essential`/`python3-dev` are used by crytic-compile's python dependencies
RUN apt-get update && \
    apt-get upgrade -y && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-suggests --no-install-recommends \
    python3 python3-pip python-is-python3 python3-dev build-essential \
    curl # `curl` is used to download nvm and golang dependencies \ 
    && rm -rf /var/lib/apt/lists/*

# Upgrade pip
RUN pip3 install --no-cache-dir --upgrade pip

# Use pip3 to install the Crytic Compile and solc-select packages
RUN pip3 install --no-cache-dir crytic-compile solc-select

# Create a directory for the Node Version Manager (nvm) and set the NVM_DIR
# environment variable
RUN mkdir /usr/local/nvm
ENV NVM_DIR /usr/local/nvm

# Download and install nvm, then use it to install the specified version of
# Node.js (the LTS version, by default). Set the installed version as the default
# and use it by default. Then install the Truffle and Hardhat Ethereum
# development tools using npm
RUN curl https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.1/install.sh | bash \
    && . $NVM_DIR/nvm.sh \
    && nvm install $NODE_VERSION --lts \
    && nvm alias default $NODE_VERSION \
    && nvm use default \
    && npm install -g truffle@$TRUFFLE_VERSION \ 
    && npm install -g hardhat@$HARDHAT_VERSION \ 
    && npm cache clean --force

# Update the PATH environment variable to include the node binary directory
ENV PATH=$PATH:/usr/local/nvm/versions/node/v$NODE_VERSION/bin/

# Use the solc-select tool to install the specified version of the Solidity
# compiler and set it as the default version
RUN solc-select install $SOLC_VERSION && solc-select use $SOLC_VERSION

# Download the Go tarball from Google's servers, extract it to the /usr/local
# directory, and remove the downloaded tarball
RUN curl -O https://dl.google.com/go/go$GO_VERSION.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go$GO_VERSION.linux-amd64.tar.gz && \
    rm -f go$GO_VERSION.linux-amd64.tar.gz

# Update the PATH environment variable to include the Go binary directory
ENV PATH=$PATH:/usr/local/go/bin

# Set the working directory to /medusa
WORKDIR /medusa

# Copy the contents of the current directory (where the Dockerfile is located)
# into the working directory in the image
COPY ../ .
