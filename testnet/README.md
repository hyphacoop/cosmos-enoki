
# Enoki Public Testnet

The Enoki public testnet provides a test environment for the current Enoki release.

* **Chain ID**: `test-enoki-1`
* **Denom**: `uoki`
* **Current version**: [`v1.7.0`](https://github.com/hyphacoop/cosmos-enoki/releases/tag/v1.7.0)
* **Genesis file:**  [genesis.json](genesis.json), verify with `shasum -a 256 genesis.json`
* **Genesis sha256sum**: `c7eac9941f3a387306f2b3b026ddd7a887ee351b60e254828a6268068ea03701`

## How to Join

The [script](./join-enoki.sh) provided in this repo will install an Enoki service on your machine.
* The script must be run either as root or from a sudoer account.
* The script will build a binary from the cosmos-enoki repo.
* The script will sync via state sync.

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

## Validator Setup

Follow the steps below to create a validator in the testnet.

1. Create a self-delegation account.
```
enokid keys add validator
```
This will output an `enoki` address, its public key, and a mnemonic. Save the mnemonic in a safe place.

2. Fund the self-delegation account.
   
Go to `https://faucet.polypore.xyz/request?address=<self-delegation-account>&chain=test-enoki-1` to get some funds sent to your account. Enter the address from the previous step instead of `<self-delegation-account>`.

3. Obtain your validator public key.
```
enokid comet show-validator
{"@type":"/cosmos.crypto.ed25519.PubKey","key":"BShP2dtw02I/1SnLp/D/RBHoeEaG3NqlMkwWYZOqcug="}
```

4. Create a validator JSON (`validator.json`) file.

Replace the `pubkey` value from the `show-validator` command above and edit the other values for your needs.
```
{
  "pubkey": {"@type":"/cosmos.crypto.ed25519.PubKey","key":"BShP2dtw02I/1SnLp/D/RBHoeEaG3NqlMkwWYZOqcug="},
  "amount": "1000000uoki",
  "moniker": "my-enoki-validator",
  "identity": null,
  "website": null,
  "security": null,
  "details": null,
  "commission-rate": "0.1",
  "commission-max-rate": "0.2",
  "commission-max-change-rate": "0.01",
  "min-self-delegation": "1000000"
}
```

5. Submit the `create-validator` transaction.

```bash
enokid tx staking create-validator \
validator.json \
--from <self-delegation-account> \
--gas auto \
--gas-adjustment 3 \
--gas-prices 0.001uoki \
--yes
```

6. Verify the validator was created.

You can confirm the validator was created with the following command:
```
enokid q staking validators -o json | jq '.validators[] | select(.consensus_pubkey.value=="<pubkey value>")'
```
Using the example above, we would query the validator with:
```
enokid q staking validators -o json | jq '.validators[] | select(.consensus_pubkey.value=="BShP2dtw02I/1SnLp/D/RBHoeEaG3NqlMkwWYZOqcug=")'
```
