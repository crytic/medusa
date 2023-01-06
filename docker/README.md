To build main medusa Docker image:
```sh
$ docker build -t medusa . --target=medusa && docker run --entrypoint /bin/bash --rm -it medusa
```

To run unit tests in medusa and copy log to the current directory:
```sh
docker build -t medusa . --target save_log --output type=local,dest=.
```

You can differentiate dependencies versions using the `--build-arg` argument:
```sh
docker build -t medusa . --target=medusa --build-arg GO_VERSION=1.18 && docker run --entrypoint /bin/sh --rm -it medusa
```

You have control over the following deps versions:
```Docker
ARG NODE_VERSION=18.13.0
ARG SOLC_VERSION=0.8.17
ARG GO_VERSION=1.19.4
ARG HARDHAT_VERSION=latest
ARG TRUFFLE_VERSION=latest
```