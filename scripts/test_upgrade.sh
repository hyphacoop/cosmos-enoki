#!/bin/bash
# This script tests upgrading from v1.5.0 to v1.6.0
# It builds both binaries, starts a chain with the old version, 
# and upgrades via governance proposal

set -eu

# Configuration
export KEY="acc0"
export CHAIN_ID="test-upgrade-1"
export MONIKER="upgrade-validator"
export KEYRING="test"
export HOME_DIR=$(eval echo "${HOME_DIR:-".enoki-upgrade"}")
export DENOM="uoki"
export PRICE="0.005"
export OLD_VERSION="v1.5.0"
export NEW_VERSION="v1.6.0"
export UPGRADE_HEIGHT=${UPGRADE_HEIGHT:-20}
export RPC_PORT=${RPC_PORT:-36657}
export P2P_PORT=${P2P_PORT:-36656}
export GRPC_PORT=${GRPC_PORT:-9092}
export PPROF_PORT=${PPROF_PORT:-9091}
export API_PORT=${API_PORT:-2317}

# Binary names
export OLD_BINARY="enokid-$OLD_VERSION"
export NEW_BINARY="enokid-$NEW_VERSION"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

echo_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

echo_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Check dependencies
command -v jq > /dev/null 2>&1 || { echo_error "jq not installed. More info: https://stedolan.github.io/jq/download/"; exit 1; }
command -v git > /dev/null 2>&1 || { echo_error "git not installed."; exit 1; }

# Clean previous test data
if [ -d "$HOME_DIR" ]; then
    echo_warning "Cleaning previous test data at $HOME_DIR"
    rm -rf "$HOME_DIR"
fi

# Build old version binary
echo_info "Building $OLD_VERSION binary..."
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
STASH_RESULT=$(git stash)

# Build v1.5.0
git checkout tags/$OLD_VERSION 2>/dev/null || git checkout $OLD_VERSION 2>/dev/null || {
    echo_error "Failed to checkout $OLD_VERSION. Make sure the tag exists."
    echo_info "Available tags:"
    git tag -l
    exit 1
}
make install
if [ ! -f "$(go env GOPATH)/bin/enokid" ]; then
    echo_error "Failed to build $OLD_VERSION binary"
    exit 1
fi
cp "$(go env GOPATH)/bin/enokid" "$(go env GOPATH)/bin/$OLD_BINARY"
echo_info "Built $OLD_BINARY successfully"

# Build new version binary
echo_info "Building $NEW_VERSION binary..."
git checkout tags/$NEW_VERSION 2>/dev/null || git checkout $NEW_VERSION 2>/dev/null || {
    echo_error "Failed to checkout $NEW_VERSION. Make sure the tag exists."
    echo_info "Available tags:"
    git tag -l
    exit 1
}
make install
if [ ! -f "$(go env GOPATH)/bin/enokid" ]; then
    echo_error "Failed to build $NEW_VERSION binary"
    exit 1
fi
cp "$(go env GOPATH)/bin/enokid" "$(go env GOPATH)/bin/$NEW_BINARY"
echo_info "Built $NEW_BINARY successfully"

# Return to original branch
git checkout $CURRENT_BRANCH
if [[ $STASH_RESULT != "No local changes to save" ]]; then
    git stash pop
fi

# Initialize chain with old binary
echo_info "Initializing chain with $OLD_BINARY..."
$OLD_BINARY config set client chain-id $CHAIN_ID --home $HOME_DIR
$OLD_BINARY config set client keyring-backend $KEYRING --home $HOME_DIR
$OLD_BINARY init $MONIKER --chain-id $CHAIN_ID --home $HOME_DIR

# Update ports in config.toml
echo_info "Configuring ports (RPC: $RPC_PORT, P2P: $P2P_PORT, pprof: $PPROF_PORT)..."
sed -i.bak "s/laddr = \"tcp:\/\/127.0.0.1:26657\"/laddr = \"tcp:\/\/127.0.0.1:$RPC_PORT\"/g" $HOME_DIR/config/config.toml
sed -i "s/laddr = \"tcp:\/\/0.0.0.0:26656\"/laddr = \"tcp:\/\/0.0.0.0:$P2P_PORT\"/g" $HOME_DIR/config/config.toml
sed -i "s/pprof_laddr = \"localhost:6060\"/pprof_laddr = \"localhost:$PPROF_PORT\"/g" $HOME_DIR/config/config.toml

# Update ports in app.toml
echo_info "Configuring app ports (GRPC: $GRPC_PORT, API: $API_PORT)..."
sed -i.bak "s/address = \"localhost:9090\"/address = \"localhost:$GRPC_PORT\"/g" $HOME_DIR/config/app.toml
sed -i "s/address = \"tcp:\/\/localhost:1317\"/address = \"tcp:\/\/localhost:$API_PORT\"/g" $HOME_DIR/config/app.toml

# Configure genesis
echo_info "Configuring genesis..."
GENESIS="$HOME_DIR/config/genesis.json"

# Update staking denom
jq --arg denom "$DENOM" '.app_state.staking.params.bond_denom = $denom' $GENESIS > temp.json && mv temp.json $GENESIS
jq --arg denom "$DENOM" '.app_state.crisis.constant_fee.denom = $denom' $GENESIS > temp.json && mv temp.json $GENESIS
jq --arg denom "$DENOM" '.app_state.gov.params.min_deposit[0].denom = $denom' $GENESIS > temp.json && mv temp.json $GENESIS
jq --arg denom "$DENOM" '.app_state.mint.params.mint_denom = $denom' $GENESIS > temp.json && mv temp.json $GENESIS

# Update feemarket denom
jq --arg denom "$DENOM" '.app_state.feemarket.params.fee_denom = $denom' $GENESIS > temp.json && mv temp.json $GENESIS
jq --arg price "$PRICE" '.app_state.feemarket.params.min_base_gas_price = $price' $GENESIS > temp.json && mv temp.json $GENESIS
jq --arg price "$PRICE" '.app_state.feemarket.state.base_gas_price = $price' $GENESIS > temp.json && mv temp.json $GENESIS


# Update governance params for faster voting
jq '.app_state.gov.params.voting_period = "30s"' $GENESIS > temp.json && mv temp.json $GENESIS
jq '.app_state.gov.params.max_deposit_period = "20s"' $GENESIS > temp.json && mv temp.json $GENESIS
jq '.app_state.gov.params.min_deposit[0].amount = "1000000"' $GENESIS > temp.json && mv temp.json $GENESIS
jq '.app_state.gov.params.expedited_voting_period = "20s"' $GENESIS > temp.json && mv temp.json $GENESIS

# Create test account
echo_info "Creating test account..."
echo "decorate bright ozone fork gallery riot bus exhaust worth way bone indoor calm squirrel merry zero scheme cotton until shop any excess stage laundry" | \
    $OLD_BINARY keys add $KEY --recover --keyring-backend $KEYRING --home $HOME_DIR

# Add genesis account
$OLD_BINARY genesis add-genesis-account $KEY 100000000000${DENOM} --keyring-backend $KEYRING --home $HOME_DIR

# Create gentx
echo_info "Creating genesis transaction..."
$OLD_BINARY genesis gentx $KEY 1000000000${DENOM} --chain-id $CHAIN_ID --keyring-backend $KEYRING --home $HOME_DIR

# Collect gentxs
$OLD_BINARY genesis collect-gentxs --home $HOME_DIR

# Validate genesis
$OLD_BINARY genesis validate-genesis --home $HOME_DIR

# Start chain in background
echo_info "Starting chain with $OLD_BINARY..."
$OLD_BINARY start --home $HOME_DIR --minimum-gas-prices="0${DENOM}" &
CHAIN_PID=$!

# Function to cleanup on exit
cleanup() {
    echo_info "Cleaning up..."
    kill $CHAIN_PID 2>/dev/null || true
    wait $CHAIN_PID 2>/dev/null || true
}
trap cleanup EXIT

# Wait for chain to start
echo_info "Waiting for chain to start..."
sleep 10

# Check if chain is running
if ! ps -p $CHAIN_PID > /dev/null; then
    echo_error "Chain failed to start"
    exit 1
fi

# Wait for first block
echo_info "Waiting for first block..."
for i in {1..30}; do
    BLOCK_HEIGHT=$($OLD_BINARY status --home $HOME_DIR 2>/dev/null | jq -r '.sync_info.latest_block_height' 2>/dev/null || echo "0")
    if [ "$BLOCK_HEIGHT" != "0" ] && [ "$BLOCK_HEIGHT" != "" ]; then
        echo_info "Chain is producing blocks. Current height: $BLOCK_HEIGHT"
        break
    fi
    sleep 1
done

if [ "$BLOCK_HEIGHT" == "0" ] || [ "$BLOCK_HEIGHT" == "" ]; then
    echo_error "Chain failed to produce blocks"
    exit 1
fi

# Verify key exists in keyring
echo_info "Verifying key '$KEY' exists in keyring..."
$OLD_BINARY keys show $KEY --keyring-backend $KEYRING --home $HOME_DIR
if [ $? -ne 0 ]; then
    echo_error "Key '$KEY' not found in keyring"
    echo_info "Available keys:"
    $OLD_BINARY keys list --keyring-backend $KEYRING --home $HOME_DIR
    exit 1
fi

# Query the governance module authority address
echo_info "Querying governance module authority..."
GOV_AUTHORITY=$($OLD_BINARY query auth module-account gov --home $HOME_DIR --output json | jq -r '.account.value.address')
echo_info "Gov module authority: $GOV_AUTHORITY"

# Create upgrade proposal JSON
echo_info "Creating upgrade proposal JSON..."
PROPOSAL_FILE="$HOME_DIR/upgrade_proposal.json"
cat > $PROPOSAL_FILE <<EOF
{
  "messages": [
    {
      "@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
      "authority": "$GOV_AUTHORITY",
      "plan": {
        "name": "$NEW_VERSION",
        "height": "$UPGRADE_HEIGHT",
        "info": "{\"binaries\":{\"linux/amd64\":\"$NEW_BINARY\"}}"
      }
    }
  ],
  "metadata": "ipfs://CID",
  "deposit": "10000000${DENOM}",
  "title": "Upgrade to $NEW_VERSION",
  "summary": "Test upgrade from $OLD_VERSION to $NEW_VERSION"
}
EOF

# Submit upgrade proposal
echo_info "Submitting upgrade proposal for $NEW_VERSION at height $UPGRADE_HEIGHT..."
$OLD_BINARY tx gov submit-proposal $PROPOSAL_FILE \
    --from $KEY \
    --chain-id $CHAIN_ID \
    --keyring-backend $KEYRING \
    --home $HOME_DIR \
    --node http://localhost:$RPC_PORT \
    --yes \
    --gas auto \
    --gas-adjustment 1.5 \
    --gas-prices ${PRICE}${DENOM}

# Wait for proposal to be created
echo_info "Waiting for proposal to be created..."
sleep 6

# Get proposal ID
PROPOSAL_ID=$($OLD_BINARY query gov proposals --home $HOME_DIR --node http://localhost:$RPC_PORT --output json | jq -r '.proposals[-1].id')
echo_info "Proposal ID: $PROPOSAL_ID"

# Vote on proposal
echo_info "Voting on proposal..."
$OLD_BINARY tx gov vote $PROPOSAL_ID yes \
    --chain-id $CHAIN_ID \
    --keyring-backend $KEYRING \
    --home $HOME_DIR \
    --node http://localhost:$RPC_PORT \
    --chain-id $CHAIN_ID \
    --from $KEY \
    --yes \
    --gas auto \
    --gas-adjustment 1.5 \
    --gas-prices ${PRICE}${DENOM}

sleep 6

# Check vote status
echo_info "Checking proposal status..."
$OLD_BINARY query gov proposal $PROPOSAL_ID --home $HOME_DIR --node http://localhost:$RPC_PORT

# Wait for voting period to end
echo_info "Waiting for voting period to end..."
sleep 30

# Check if proposal passed
$OLD_BINARY query gov proposal $PROPOSAL_ID --home $HOME_DIR --node http://localhost:$RPC_PORT --output json | jq -r '.proposal.status'
PROPOSAL_STATUS=$($OLD_BINARY query gov proposal $PROPOSAL_ID --home $HOME_DIR --node http://localhost:$RPC_PORT --output json | jq -r '.proposal.status')
echo_info "Proposal status: $PROPOSAL_STATUS"

if [ "$PROPOSAL_STATUS" != "PROPOSAL_STATUS_PASSED" ]; then
    echo_error "Proposal did not pass. Status: $PROPOSAL_STATUS"
    exit 1
fi

echo_info "Proposal passed! Waiting for upgrade height..."

# Monitor blocks until upgrade height
$OLD_BINARY status --home $HOME_DIR --node http://localhost:$RPC_PORT
while true; do
    CURRENT_HEIGHT=$($OLD_BINARY status --home $HOME_DIR --node http://localhost:$RPC_PORT | jq -r '.sync_info.latest_block_height' 2>/dev/null || echo "0")
    echo_info "Current height: $CURRENT_HEIGHT, Upgrade height: $UPGRADE_HEIGHT"
    
    if [ "$CURRENT_HEIGHT" -ge "$((UPGRADE_HEIGHT - 1))" ]; then
        echo_info "Approaching upgrade height..."
        break
    fi
    sleep 2
done

# Wait for chain to halt
echo_info "Waiting for chain to halt at upgrade height..."
sleep 10

# Check if old binary stopped
if ps -p $CHAIN_PID > /dev/null; then
    echo_warning "Old binary still running, stopping it..."
    kill $CHAIN_PID 2>/dev/null || true
    wait $CHAIN_PID 2>/dev/null || true
fi

# Start new binary
echo_info "Starting chain with $NEW_BINARY..."
$NEW_BINARY start --home $HOME_DIR --minimum-gas-prices="0${DENOM}" &
NEW_CHAIN_PID=$!

# Update cleanup trap
cleanup() {
    echo_info "Cleaning up..."
    kill $NEW_CHAIN_PID 2>/dev/null || true
    wait $NEW_CHAIN_PID 2>/dev/null || true
}
trap cleanup EXIT

# Wait for new binary to start producing blocks
echo_info "Waiting for upgraded chain to produce blocks..."
sleep 5

# Verify upgrade
for i in {1..30}; do
    NEW_HEIGHT=$($NEW_BINARY status --home $HOME_DIR --node http://localhost:$RPC_PORT 2>/dev/null | jq -r '.sync_info.latest_block_height' 2>/dev/null || echo "0")
    if [ "$NEW_HEIGHT" -gt "$UPGRADE_HEIGHT" ]; then
        echo_info "Upgrade successful! New chain height: $NEW_HEIGHT"
        echo_info "✅ Upgrade test completed successfully!"
        
        # Query account to verify chain is functional
        $NEW_BINARY query bank balances $($NEW_BINARY keys show $KEY -a --keyring-backend $KEYRING --home $HOME_DIR) --home $HOME_DIR --node http://localhost:$RPC_PORT
        
        exit 0
    fi
    sleep 2
done

echo_error "Upgrade failed - chain did not produce blocks after upgrade"
exit 1
