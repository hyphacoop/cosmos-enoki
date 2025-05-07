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

## Building ENoki

### Local Builds

- `make install`      *Builds the chain's binary*
- `docker build -t enoki:local`  *Builds the chain's docker image*

### Local Testnet

- `make sh-testnet` *Single node, no IBC. quick iteration*
- `make testnet` *IBC testnet from chain <-> local cosmos-hub*
- `local-ic chains` *See available testnets from the chains/ directory*
- `local-ic start <name>` *Starts a local chain with the given name*

### Testing

- `go test ./... -v` *Unit test*
- `make ictest-*`  *E2E testing*

