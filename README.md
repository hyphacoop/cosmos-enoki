[![Join Enoki Testnet](https://github.com/hyphacoop/cosmos-enoki/actions/workflows/join-testnet.yml/badge.svg)](https://github.com/hyphacoop/cosmos-enoki/actions/workflows/join-testnet.yml)

# Enoki 🍄

Enoki is Hypha's reference binary for the Interchain stack. There is a [public testnet available](/testnet/README.md)!

## Features

* Cosmos SDK v0.53.2
* CometBFT v0.38.17
* IBC v10.3.0
* Modules
  * Circuit
  * Group
  * Rate Limit
  * Token factory
  * Feemarket v1.1.1
  * Wasmd v0.60.1
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

