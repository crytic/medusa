## medusa build process
FROM golang:1.22 AS medusa

WORKDIR /src
COPY . /src/medusa/
RUN cd medusa && \
    go build -trimpath -o=/usr/local/bin/medusa -ldflags="-s -w" && \
    chmod 755 /usr/local/bin/medusa


## Python dependencies
FROM ubuntu:noble AS builder-python3
RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-suggests --no-install-recommends \
        gcc \
        python3 \
        python3-dev \
        python3-venv
ENV PIP_DISABLE_PIP_VERSION_CHECK=1
ENV PIP_NO_CACHE_DIR=1
RUN python3 -m venv /venv && /venv/bin/pip3 install --no-cache --upgrade setuptools pip
RUN /venv/bin/pip3 install --no-cache slither-analyzer solc-select


## final image assembly
FROM ubuntu:noble AS final-ubuntu

RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-suggests --no-install-recommends \
        ca-certificates \
        curl \
        git \
        jq \
        python3 \
        && \
    rm -rf /var/lib/apt/lists/*

# Include python tools
COPY --from=builder-python3 /venv /venv
ENV PATH="$PATH:/venv/bin"

# Include JS package managers, actions/setup-node can get confused if they're not around
RUN curl -fsSL https://raw.githubusercontent.com/tj/n/v10.1.0/bin/n -o n && \
    if [ ! "a09599719bd38af5054f87b8f8d3e45150f00b7b5675323aa36b36d324d087b9  n" = "$(sha256sum n)" ]; then \
        echo "N installer does not match expected checksum! exiting"; \
        exit 1; \
    fi && \
    cat n | bash -s lts && rm n && \
    npm install -g n yarn && \
    n stable --cleanup && n prune && npm --force cache clean

# Include medusa
COPY --chown=root:root --from=medusa /usr/local/bin/medusa /usr/local/bin/medusa
