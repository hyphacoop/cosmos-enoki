[![Join Enoki Testnet](https://github.com/hyphacoop/cosmos-enoki/actions/workflows/join-testnet.yml/badge.svg)](https://github.com/hyphacoop/cosmos-enoki/actions/workflows/join-testnet.yml)

# Enoki 🍄

Enoki is Hypha's reference binary for the Interchain stack. There is a [public testnet available](/testnet/README.md)!

## Features

* Modules
  * Circuit
  * tokenfactory
  * Feemarket
  * Wasmd
* Ledger support

#### Version Selection

* The module versions in this build are considered to be stable.
* Dependencies will be bumped as the compatibility between modules allows.

## Local Builds

- `make install`
  - Builds the chain's binary
- `docker build -t enoki:local .`
  - Builds the chain's Docker image

## [Public Testnet](/testnet/README.md)

## Local Testnet

- `make sh-testnet`
  - Single shell-based node, no IBC
- `make ic-testnet`
  - Interchaintest testnet: single Enoki chain
- `make ic-testnet-ibc`
  - Interchaintest testnet: Enoki chain 1 <--IBC--> Enoki chain 2
- `make ic-testnet-gaia`
  - Interchaintest testnet: Enoki chain <--IBC--> Gaia chain

