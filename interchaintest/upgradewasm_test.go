package e2e

// Upgrade CosmWasm Test Suite
//
// This test validates that CosmWasm contracts continue to work correctly
// through a chain software upgrade from v1.6.0 to v1.7.0.
//
// Prerequisites:
//   1. Docker images must be built locally before running these tests:
//      - docker build -t enoki:v1.6.0 . (from v1.6.0 commit/tag)
//      - docker build -t enoki:v1.7.0 . (from v1.7.0 commit/tag)
//
//   2. Docker daemon must be running and accessible
//
//   3. Ensure Docker has sufficient resources (CPU, memory, disk space)
//
// Usage:
//   go test -v -run TestChainUpgradeCosmWasm -timeout 15m

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestChainUpgradeCosmWasm tests that CosmWasm contracts work before and after a chain upgrade
func TestChainUpgradeCosmWasm(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)
	client, network := interchaintest.DockerSetup(t)

	// Create a chain spec with v1.6.0 as the pre-upgrade version
	preUpgradeVersion := "v1.7.0"
	postUpgradeVersion := "v1.8.0"

	preUpgradeImage := ibc.NewDockerImage("enoki", preUpgradeVersion, "1025:1025")
	postUpgradeImage := ibc.NewDockerImage("enoki", postUpgradeVersion, "1025:1025")

	chainSpec := &DefaultChainSpec
	chainSpec.Version = preUpgradeVersion
	// Include both images so interchaintest doesn't try to pull them
	chainSpec.ChainConfig.Images = []ibc.DockerImage{preUpgradeImage, postUpgradeImage}

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		chainSpec,
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	chain := chains[0].(*cosmos.CosmosChain)

	// Setup Interchain
	ic := interchaintest.NewInterchain().
		AddChain(chain)

	require.NoError(t, ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:          t.Name(),
		Client:            client,
		NetworkID:         network,
		SkipPathCreation:  false,
		BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})

	// Create and fund test user
	amt := math.NewInt(100_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", amt, chain)
	user := users[0]

	// Verify chain is running pre-upgrade version
	height, err := chain.Height(ctx)
	require.NoError(t, err)
	require.Greater(t, height, int64(0), "chain should be running")

	// ========================================
	// PRE-UPGRADE: Deploy and test contract
	// ========================================
	t.Log("📝 PRE-UPGRADE: Deploying and testing CosmWasm contract...")

	codeId, contractAddr := SetupContract(t, ctx, chain, user.KeyName(), "contracts/cw_template.wasm", `{"count":0}`)
	t.Logf("Contract deployed at: %s (code_id: %s)", contractAddr, codeId)

	// Execute contract before upgrade
	_, err = chain.ExecuteContract(ctx, user.KeyName(), contractAddr, `{"increment":{}}`, "--fees", "10000"+chain.Config().Denom)
	require.NoError(t, err)

	// Query contract state before upgrade
	var preUpgradeRes GetCountResponse
	err = SmartQueryString(t, ctx, chain, contractAddr, `{"get_count":{}}`, &preUpgradeRes)
	require.NoError(t, err)
	require.Equal(t, int64(1), preUpgradeRes.Data.Count)
	t.Logf("✅ Pre-upgrade contract state: count = %d", preUpgradeRes.Data.Count)

	// ========================================
	// UPGRADE: Submit and execute upgrade
	// ========================================
	t.Log("⚙️  Submitting upgrade proposal...")

	upgradeHeight := haltHeight
	upgradeName := postUpgradeVersion

	// Perform upgrade using helper functions
	params := UpgradeParams{
		UpgradeName:        upgradeName,
		UpgradeHeight:      upgradeHeight,
		PostUpgradeVersion: postUpgradeVersion,
	}

	PerformUpgrade(ctx, t, chain, client, user.KeyName(), params)

	// ========================================
	// POST-UPGRADE: Verify contract still works
	// ========================================
	t.Log("📝 POST-UPGRADE: Testing existing CosmWasm contract...")

	// Query contract state after upgrade (should still be 1)
	var postUpgradeRes GetCountResponse
	err = SmartQueryString(t, ctx, chain, contractAddr, `{"get_count":{}}`, &postUpgradeRes)
	require.NoError(t, err)
	require.Equal(t, int64(1), postUpgradeRes.Data.Count)
	t.Logf("✅ Post-upgrade contract state persisted: count = %d", postUpgradeRes.Data.Count)

	// Execute contract after upgrade
	_, err = chain.ExecuteContract(ctx, user.KeyName(), contractAddr, `{"increment":{}}`, "--fees", "10000"+chain.Config().Denom)
	require.NoError(t, err)

	// Query contract state after execution (should be 2)
	var finalRes GetCountResponse
	err = SmartQueryString(t, ctx, chain, contractAddr, `{"get_count":{}}`, &finalRes)
	require.NoError(t, err)
	require.Equal(t, int64(2), finalRes.Data.Count)
	t.Logf("✅ Post-upgrade contract execution successful: count = %d", finalRes.Data.Count)

	// Deploy a new contract after upgrade to verify wasm module still works
	t.Log("📝 POST-UPGRADE: Deploying new CosmWasm contract...")
	_, newContractAddr := SetupContract(t, ctx, chain, user.KeyName(), "contracts/cw_template.wasm", `{"count":10}`)
	t.Logf("New contract deployed at: %s", newContractAddr)

	// Test the new contract
	_, err = chain.ExecuteContract(ctx, user.KeyName(), newContractAddr, `{"increment":{}}`, "--fees", "10000"+chain.Config().Denom)
	require.NoError(t, err)

	var newContractRes GetCountResponse
	err = SmartQueryString(t, ctx, chain, newContractAddr, `{"get_count":{}}`, &newContractRes)
	require.NoError(t, err)
	require.Equal(t, int64(11), newContractRes.Data.Count)
	t.Logf("✅ New contract works correctly: count = %d", newContractRes.Data.Count)

	t.Log("✅ CosmWasm upgrade test completed successfully")
}
