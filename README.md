# Generated With [Spawn](https://github.com/rollchains/spawn)

## Testnet

- `make testnet` *IBC testnet from chain <-> local cosmos-hub*
- `make sh-testnet` *Single node, no IBC. quick iteration*
- `local-ic chains` *See available testnets from the chains/ directory*
- `local-ic start <name>` *Starts a local chain with the given name*

## Local Images

- `make install`      *Builds the chain's binary*
- `make local-image`  *Builds the chain's docker image*

## Testing

- `go test ./... -v` *Unit test*
- `make ictest-*`  *E2E testing*

