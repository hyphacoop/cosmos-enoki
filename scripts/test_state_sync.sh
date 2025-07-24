#!/bin/bash
# Run this script to quickly install, setup, and run the current version of the network without docker.
#
# Examples:
# CHAIN_ID="localchain-1" HOME_DIR="~/.enoki" BLOCK_TIME="1000ms" CLEAN=true sh scripts/test_node.sh
# CHAIN_ID="localchain-2" HOME_DIR="~/.enoki" CLEAN=true RPC=36657 REST=2317 PROFF=6061 P2P=36656 GRPC=8090 GRPC_WEB=8091 ROSETTA=8081 BLOCK_TIME="500ms" sh scripts/test_node.sh

set -eu

export KEY="acc0"
export KEY2="acc1"

export CHAIN_ID=${CHAIN_ID:-"test-enoki-1"}
export MONIKER="statesync"
export KEYALGO="secp256k1"
export KEYRING=${KEYRING:-"test"}
export HOME_DIR=$(eval echo "${HOME_DIR:-".statesync"}")
export BINARY=${BINARY:-enokid}
export DENOM=${DENOM:-uoki}

export CLEAN=${CLEAN:-"false"}
export RPC=${RPC:-"37657"}
export REST=${REST:-"2417"}
export PROFF=${PROFF:-"7160"}
export P2P=${P2P:-"37656"}
export GRPC=${GRPC:-"9290"}
export GRPC_WEB=${GRPC_WEB:-"9291"}
export ROSETTA=${ROSETTA:-"8280"}
export BLOCK_TIME=${BLOCK_TIME:-"2s"}
export SYNC_RPC='http://localhost:26657'
export SYNC_RPC_SERVERS="$SYNC_RPC,$SYNC_RPC"
export TRUST_OFFSET=100
export SNAPSHOT_INTERVAL=${SNAPSHOT_INTERVAL:-100}

set_config() {
  $BINARY config set client chain-id $CHAIN_ID --home $HOME_DIR
  $BINARY config set client keyring-backend $KEYRING --home $HOME_DIR
}
rm -rf $HOME_DIR && echo "Removed $HOME_DIR"

$BINARY init statesync --chain-id $CHAIN_ID --default-denom $DENOM --home $HOME_DIR
set_config
cp .enoki/config/genesis.json .statesync/config/genesis.json

echo "Starting node..."

echo "> Open the RPC endpoint to outside connections"
sed -i -e 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:'$RPC'"|g' $HOME_DIR/config/config.toml
sed -i -e 's|cors_allowed_origins = \[\]|cors_allowed_origins = \["*"\]|g' $HOME_DIR/config/config.toml

echo "> Set up REST endpoint"
sed -i -e 's|address = "tcp://localhost:1317"|address = "tcp://0.0.0.0:'$REST'"|g' $HOME_DIR/config/app.toml
sed -i -e 's/enable = false/enable = true/g' $HOME_DIR/config/app.toml
sed -i -e 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' $HOME_DIR/config/app.toml

# peer exchange
sed -i -e 's|pprof_laddr = "localhost:6060"|pprof_laddr = "localhost:'$PROFF'"|g' $HOME_DIR/config/config.toml
sed -i -e 's|laddr = "tcp://0.0.0.0:26656"|laddr = "tcp://0.0.0.0:'$P2P'"|g' $HOME_DIR/config/config.toml

# GRPC
sed -i -e 's/address = "localhost:9090"/address = "0.0.0.0:'$GRPC'"/g' $HOME_DIR/config/app.toml
sed -i -e 's/address = "localhost:9091"/address = "0.0.0.0:'$GRPC_WEB'"/g' $HOME_DIR/config/app.toml

# Rosetta Api
sed -i -e 's/address = ":8080"/address = "0.0.0.0:'$ROSETTA'"/g' $HOME_DIR/config/app.toml

echo "> Set timeout commit"
sed -i -e 's/timeout_commit = "5s"/timeout_commit = "'$BLOCK_TIME'"/g' $HOME_DIR/config/config.toml

echo "> Populate peers"
peer_id=$($BINARY comet show-node-id --home .enoki)
peer="$peer_id@127.0.0.1:26656"
sed -i -e 's|persistent_peers = ""|persistent_peers = "'$peer'"|g' $HOME_DIR/config/config.toml

# Enable duplicate IP connections
sed -i -e 's|allow_duplicate_ip = false|allow_duplicate_ip = true|g' $HOME_DIR/config/config.toml

echo "> Enable state sync"
CURRENT_BLOCK=$(curl -s $SYNC_RPC/block | jq -r '.result.block.header.height')
echo "> Current block: $CURRENT_BLOCK"
TRUST_HEIGHT=$[ $CURRENT_BLOCK-$TRUST_OFFSET ]
echo "> Trust height: $TRUST_HEIGHT"
TRUST_BLOCK=$(curl -s $SYNC_RPC/block\?height\=$TRUST_HEIGHT)
echo "> Update config again"
TRUST_HASH=$(echo $TRUST_BLOCK | jq -r '.result.block_id.hash')

# sed -i -e '/enable =/ s/= .*/= true/' $HOME_DIR/config/config.toml
sed -i -e '/trust_period =/ s/= .*/= "24h0m0s"/' $HOME_DIR/config/config.toml
sed -i -e "/trust_height =/ s/= .*/= $TRUST_HEIGHT/" $HOME_DIR/config/config.toml
sed -i -e "/trust_hash =/ s/= .*/= \"$TRUST_HASH\"/" $HOME_DIR/config/config.toml
sed -i -e "/rpc_servers =/ s^= .*^= \"$SYNC_RPC_SERVERS\"^" $HOME_DIR/config/config.toml

# Enable state sync snapshots
echo "> Snapshot internal: $SNAPSHOT_INTERVAL"
sed -i -e 's|snapshot-interval = 0|snapshot-interval = '$SNAPSHOT_INTERVAL'|g' $HOME_DIR/config/app.toml

$BINARY start --pruning=nothing --home $HOME_DIR | tee statesync.log