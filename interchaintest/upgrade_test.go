package e2e

// Upgrade Test Suite
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
// Known Issues:
//   - If you see "No such container" errors for volume-owner containers,
//     try: docker system prune -f && docker volume prune -f
//   - Ensure no stale interchaintest containers are running: docker ps -a
//
// Usage:
// go test -v -run TestChainUpgrade -timeout 10m \
//   -pre-upgrade-version v1.8.0 \
//   -post-upgrade-version v1.9.0 \
//   -upgrade-name v1.9.0
// go test -v -run TestChainUpgradeWithValidation -timeout 10m \
//   -pre-upgrade-version v1.8.0 \
//   -post-upgrade-version v1.9.0 \
//   -upgrade-name v1.9.0

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

const (
	haltHeight         = int64(50)
	blocksAfterUpgrade = int64(10)
)

// TestChainUpgrade tests a chain software upgrade
func TestChainUpgrade(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)
	client, network := interchaintest.DockerSetup(t)

	// Create a chain spec
	preUpgradeVersion := *flagPreUpgradeVersion
	postUpgradeVersion := *flagPostUpgradeVersion
	upgradeName := *flagUpgradeName

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

	// Perform upgrade using helper functions
	params := UpgradeParams{
		UpgradeName:        upgradeName,
		UpgradeHeight:      haltHeight,
		PostUpgradeVersion: postUpgradeVersion,
	}

	PerformUpgrade(ctx, t, chain, client, user.KeyName(), params)

	t.Log("✅ Upgrade test completed successfully")
}

// TestChainUpgradeWithPreAndPostUpgradeValidation tests a chain software upgrade
// with comprehensive pre and post-upgrade validation
func TestChainUpgradeWithValidation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)
	client, network := interchaintest.DockerSetup(t)

	preUpgradeVersion := *flagPreUpgradeVersion
	postUpgradeVersion := *flagPostUpgradeVersion

	preUpgradeImage := ibc.NewDockerImage("enoki", preUpgradeVersion, "1025:1025")
	postUpgradeImage := ibc.NewDockerImage("enoki", postUpgradeVersion, "1025:1025")

	chainSpec := &DefaultChainSpec
	chainSpec.Version = preUpgradeVersion
	chainSpec.ChainConfig.Images = []ibc.DockerImage{preUpgradeImage, postUpgradeImage}

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		chainSpec,
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	chain := chains[0].(*cosmos.CosmosChain)

	ic := interchaintest.NewInterchain().
		AddChain(chain)

	require.NoError(t, ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: false,
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})

	amt := math.NewInt(100_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", amt, chain)
	user := users[0]

	// Pre-upgrade validation
	t.Log("🔍 Running pre-upgrade validation...")

	// Store pre-upgrade state
	preUpgradeHeight, err := chain.Height(ctx)
	require.NoError(t, err)
	t.Logf("Pre-upgrade height: %d", preUpgradeHeight)

	preUpgradeBalance, err := chain.GetBalance(ctx, user.FormattedAddress(), Denom)
	require.NoError(t, err)
	t.Logf("Pre-upgrade user balance: %s%s", preUpgradeBalance.String(), Denom)

	// Create some state before upgrade (e.g., send tokens)
	recipient := "enoki1hj5fveer5cjtn4wd6wstzugjfdxzl0xp2w67r4"
	preUpgradeTransferAmount := math.NewInt(5_000)

	err = chain.SendFunds(ctx, user.KeyName(), ibc.WalletAmount{
		Address: recipient,
		Denom:   Denom,
		Amount:  preUpgradeTransferAmount,
	})
	require.NoError(t, err)

	recipientPreBalance, err := chain.GetBalance(ctx, recipient, Denom)
	require.NoError(t, err)
	t.Logf("Pre-upgrade recipient balance: %s%s", recipientPreBalance.String(), Denom)

	// Perform upgrade
	t.Log("⚙️  Submitting upgrade proposal...")

	upgradeHeight := haltHeight
	upgradeName := *flagUpgradeName

	upgradeParams := UpgradeParams{
		UpgradeName:        upgradeName,
		UpgradeHeight:      upgradeHeight,
		PostUpgradeVersion: postUpgradeVersion,
	}

	proposalID := SubmitUpgradeProposal(ctx, t, chain, user.KeyName(), upgradeParams)
	WaitForProposalVotingPeriod(ctx, t, chain, proposalID)
	VoteOnProposal(ctx, t, chain, proposalID, "validator")
	WaitForProposalPass(ctx, t, chain, proposalID)
	ExecuteUpgrade(ctx, t, chain, client, upgradeParams)

	// Post-upgrade validation
	t.Log("✅ Running post-upgrade validation...")

	node := chain.GetNode()
	var stdout []byte

	postUpgradeHeight, err := chain.Height(ctx)
	require.NoError(t, err)
	require.Greater(t, postUpgradeHeight, upgradeHeight)
	t.Logf("Post-upgrade height: %d", postUpgradeHeight)

	// Verify state persistence
	recipientPostBalance, err := chain.GetBalance(ctx, recipient, Denom)
	require.NoError(t, err)
	require.Equal(t, recipientPreBalance, recipientPostBalance, "recipient balance should persist through upgrade")
	t.Logf("Post-upgrade recipient balance: %s%s (maintained)", recipientPostBalance.String(), Denom)

	// Verify chain functionality after upgrade
	postUpgradeTransferAmount := math.NewInt(3_000)
	err = chain.SendFunds(ctx, user.KeyName(), ibc.WalletAmount{
		Address: recipient,
		Denom:   Denom,
		Amount:  postUpgradeTransferAmount,
	})
	require.NoError(t, err)

	finalBalance, err := chain.GetBalance(ctx, recipient, Denom)
	require.NoError(t, err)
	expectedBalance := recipientPostBalance.Add(postUpgradeTransferAmount)
	require.True(t, finalBalance.GTE(expectedBalance), "post-upgrade transfer should work")
	t.Logf("Final recipient balance after post-upgrade transfer: %s%s", finalBalance.String(), Denom)

	// Verify upgrade was applied
	stdout, _, err = node.ExecQuery(ctx, "upgrade", "applied", upgradeName)
	require.NoError(t, err)
	require.NotEmpty(t, stdout)
	t.Logf("Upgrade %s successfully applied", upgradeName)

	// Additional validation: query module parameters
	stdout, _, err = node.ExecQuery(ctx, "bank", "total")
	require.NoError(t, err)
	require.NotEmpty(t, stdout)

	t.Log("✅ All post-upgrade validations passed successfully")
}
