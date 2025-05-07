# Enoki 🍄

Enoki is Hypha's reference binary for the Interchain stack.

## Features

* Cosmos SDK v0.53.0
* CometBFT v0.38.17
* IBC v10.2.0
* Modules
  * Circuit
  * Group
  * NFT
  * Rate Limit
  * Token factory
  * Feemarket v1.1.1
  * Wasmd v0.55.0
* Ledger support

## Local Builds

- `make install`
  - Builds the chain's binary
- `docker build -t enoki:local .`
  - Builds the chain's Docker image

## Local Testnet

- `make sh-testnet`
  - Single shell-based node, no IBC
- `make ic-testnet`
  - Interchaintest testnet: single Enoki chain
- `make ic-testnet-ibc`
  - Interchaintest testnet: Enoki chain 1 <--IBC--> Enoki chain 2
- `make ic-testent-gaia`
  - Interchaintest testnet: Enoki chain <--IBC--> Gaia chain

