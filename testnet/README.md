
# Enoki Public Testnet

The Enoki public testnet provides a test environment for the current Enoki release.

* **Chain ID**: `test-enoki-1`
* **Denom**: `uoki`
* **Current version**: [`v1.0.0-rc0`](httpshttps://github.com/hyphacoop/cosmos-enoki/releases/tag/v1.0.0-rc0)
* **Genesis file:**  [genesis.json](genesis.json), verify with `shasum -a 256 genesis.json`
* **Genesis sha256sum**: `34498254ca51abe183c22ae2952649968f87cb1f7a707320ec57c9e2408db026`

## Endpoints

### Peers

* `6979df4e3fa27dcef2fe1a22ad4b8c052c423537@sentry-01.enoki.polypore.xyz:26656`

### RPC

* `https://rpc.sentry-01.enoki.polypore.xyz`

### API

* `https://api.sentry-01.enoki.polypore.xyz`

### gRPC

* `sentry-01.enoki.polypore.xyz:9090`

### State sync

1. `https://rpc.sentry-01.enoki.polypore.xyz:443`

## Faucet

* Visit `faucet.polypore.xyz` to request tokens and check your address balance.

## How to Join

The [script](./join-enoki.sh) provided in this repo will install an Enoki service on your machine.
* The script must be run either as root or from a sudoer account.
* The script will build a binary from the cosmos-enoki repo.
* The script will sync via state sync.
