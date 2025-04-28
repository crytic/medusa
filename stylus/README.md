# Medusa Stylus

This is the Stylus variant of Medusa, which is built against the Offchain Labs version of go-ethereum.

## Installation

```bash
# Install directly from GitHub
go install -tags=stylus github.com/crytic/medusa/stylus@latest
```

## Building from Source

```bash
# Clone the repository
git clone https://github.com/crytic/medusa.git
cd medusa/stylus

# Build the stylus version
go build -tags=stylus

# Run the stylus version
./stylus
```

## Differences from Standard Medusa

The Stylus version of Medusa is built against the Offchain Labs fork of go-ethereum (stylus-v1.15.5 branch),
which enables additional functionality for Arbitrum Stylus smart contracts.

All core Medusa features are available in both versions, but this version specifically targets
the Arbitrum Stylus environment.