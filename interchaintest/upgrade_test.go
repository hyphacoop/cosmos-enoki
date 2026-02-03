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
//   go test -v -run TestChainUpgrade -timeout 10m
//   go test -v -run TestChainUpgradeWithValidation -timeout 15m

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
	validatorKeyName   = "validator" // Standard validator key name in interchaintest
)

// TestChainUpgrade tests a chain software upgrade from v1.6.0 to v1.7.0
func TestChainUpgrade(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)
	client, network := interchaintest.DockerSetup(t)

	// Create a chain spec with v1.6.0 as the pre-upgrade version
	preUpgradeVersion := "v1.6.0"
	postUpgradeVersion := "v1.7.0"
	upgradeName := "v1.7.0"

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

// // TestChainUpgradeWithPreAndPostUpgradeValidation tests a chain software upgrade
// // with comprehensive pre and post-upgrade validation
// func TestChainUpgradeWithValidation(t *testing.T) {
// 	t.Parallel()

// 	ctx := context.Background()
// 	rep := testreporter.NewNopReporter()
// 	eRep := rep.RelayerExecReporter(t)
// 	client, network := interchaintest.DockerSetup(t)

// 	preUpgradeVersion := "v1.6.0"
// 	postUpgradeVersion := "v1.7.0"

// 	preUpgradeImage := ibc.NewDockerImage("enoki", preUpgradeVersion, "1025:1025")

// 	chainSpec := &DefaultChainSpec
// 	chainSpec.Version = preUpgradeVersion
// 	chainSpec.ChainConfig.Images = []ibc.DockerImage{preUpgradeImage}

// 	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
// 		chainSpec,
// 	})

// 	chains, err := cf.Chains(t.Name())
// 	require.NoError(t, err)

// 	chain := chains[0].(*cosmos.CosmosChain)

// 	ic := interchaintest.NewInterchain().
// 		AddChain(chain)

// 	require.NoError(t, ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
// 		TestName:         t.Name(),
// 		Client:           client,
// 		NetworkID:        network,
// 		SkipPathCreation: false,
// 	}))
// 	t.Cleanup(func() {
// 		_ = ic.Close()
// 	})

// 	amt := math.NewInt(100_000_000)
// 	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", amt, chain)
// 	user := users[0]

// 	// Pre-upgrade validation
// 	t.Log("🔍 Running pre-upgrade validation...")

// 	// Store pre-upgrade state
// 	preUpgradeHeight, err := chain.Height(ctx)
// 	require.NoError(t, err)
// 	t.Logf("Pre-upgrade height: %d", preUpgradeHeight)

// 	preUpgradeBalance, err := chain.GetBalance(ctx, user.FormattedAddress(), Denom)
// 	require.NoError(t, err)
// 	t.Logf("Pre-upgrade user balance: %s%s", preUpgradeBalance.String(), Denom)

// 	// Create some state before upgrade (e.g., send tokens)
// 	recipient := "enoki1hj5fveer5cjtn4wd6wstzugjfdxzl0xp2w67r4"
// 	preUpgradeTransferAmount := math.NewInt(5_000)

// 	err = chain.SendFunds(ctx, user.KeyName(), ibc.WalletAmount{
// 		Address: recipient,
// 		Denom:   Denom,
// 		Amount:  preUpgradeTransferAmount,
// 	})
// 	require.NoError(t, err)

// 	recipientPreBalance, err := chain.GetBalance(ctx, recipient, Denom)
// 	require.NoError(t, err)
// 	t.Logf("Pre-upgrade recipient balance: %s%s", recipientPreBalance.String(), Denom)

// 	// Perform upgrade
// 	t.Log("⚙️  Submitting upgrade proposal...")

// 	upgradeHeight := haltHeight
// 	upgradeName := postUpgradeVersion

// 	node := chain.GetNode()

// 	// Query gov params to determine minimum deposit
// 	t.Log("🔍 Querying governance parameters...")
// 	govParamsCmd := []string{chain.Config().Bin, "query", "gov", "params", "--node", chain.GetRPCAddress(), "--output", "json"}
// 	stdout, _, err := node.Exec(ctx, govParamsCmd, nil)
// 	require.NoError(t, err)

// 	// Parse the params JSON manually to extract min_deposit
// 	var paramsResult map[string]interface{}
// 	err = json.Unmarshal(stdout, &paramsResult)
// 	require.NoError(t, err)
// 	params, ok := paramsResult["params"].(map[string]interface{})
// 	require.True(t, ok, "params field should exist")
// 	minDepositArray, ok := params["min_deposit"].([]interface{})
// 	require.True(t, ok && len(minDepositArray) > 0, "min_deposit should be a non-empty array")
// 	minDepositCoin := minDepositArray[0].(map[string]interface{})
// 	depositAmount := fmt.Sprintf("%s%s", minDepositCoin["amount"].(string), minDepositCoin["denom"].(string))
// 	t.Logf("Minimum deposit required: %s", depositAmount)

// 	// Query feemarket for gas prices
// 	t.Log("🔍 Querying feemarket gas prices...")
// 	gasPricesCmd := []string{chain.Config().Bin, "query", "feemarket", "gas-prices", "--node", chain.GetRPCAddress(), "--output", "json"}
// 	stdout, _, err = node.Exec(ctx, gasPricesCmd, nil)
// 	require.NoError(t, err)

// 	var gasPricesResult map[string]interface{}
// 	err = json.Unmarshal(stdout, &gasPricesResult)
// 	require.NoError(t, err)
// 	pricesArray, ok := gasPricesResult["prices"].([]interface{})
// 	require.True(t, ok && len(pricesArray) > 0, "prices should be a non-empty array")
// 	priceObj := pricesArray[0].(map[string]interface{})
// 	gasPrice := fmt.Sprintf("%s%s", priceObj["amount"].(string), priceObj["denom"].(string))
// 	t.Logf("Gas price: %s", gasPrice)

// 	// Create the proposal content
// 	authority := "enoki10d07y265gmmuvt4z0w9aw880jnsr700jkqmfe9" // gov module authority
// 	proposalJSON := fmt.Sprintf(`{
// 		"messages": [{
// 			"@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
// 			"authority": "%s",
// 			"plan": {
// 				"name": "%s",
// 				"height": "%d",
// 				"info": "Upgrade to %s"
// 			}
// 		}],
// 		"metadata": "ipfs://CID",
// 		"deposit": "%s",
// 		"title": "Upgrade to %s",
// 		"summary": "Upgrade chain to %s at height %d"
// 	}`, authority, upgradeName, upgradeHeight, upgradeName, depositAmount, upgradeName, upgradeName, upgradeHeight)

// 	// Write proposal JSON to file in node's home directory
// 	proposalFile := node.HomeDir() + "/upgrade-proposal.json"
// 	writeCmd := []string{"sh", "-c", fmt.Sprintf("echo '%s' > %s", proposalJSON, proposalFile)}
// 	_, _, err = node.Exec(ctx, writeCmd, nil)
// 	require.NoError(t, err)

// 	// Submit proposal
// 	submitCmd := []string{
// 		chain.Config().Bin,
// 		"tx", "gov", "submit-proposal", proposalFile,
// 		"--from", user.KeyName(),
// 		"--chain-id", chain.Config().ChainID,
// 		"--node", chain.GetRPCAddress(),
// 		"--home", node.HomeDir(),
// 		"--keyring-backend", "test",
// 		"--gas", "auto",
// 		"--gas-adjustment", "2.0",
// 		"--gas-prices", gasPrice,
// 		"--yes",
// 		"--output", "json",
// 	}

// 	stdout, _, err = node.Exec(ctx, submitCmd, nil)
// 	require.NoError(t, err)

// 	err = testutil.WaitForBlocks(ctx, 2, chain)
// 	require.NoError(t, err)

// 	// Extract transaction hash from submission output
// 	var txResponse map[string]interface{}
// 	err = json.Unmarshal(stdout, &txResponse)
// 	require.NoError(t, err)
// 	txHash, ok := txResponse["txhash"].(string)
// 	require.True(t, ok && txHash != "", "transaction hash should be present")
// 	t.Logf("Transaction hash: %s", txHash)

// 	// Query transaction to extract proposal ID from events
// 	txQueryCmd := []string{chain.Config().Bin, "query", "tx", txHash, "--node", chain.GetRPCAddress(), "--output", "json"}
// 	stdout, _, err = node.Exec(ctx, txQueryCmd, nil)
// 	require.NoError(t, err)

// 	var txResult map[string]interface{}
// 	err = json.Unmarshal(stdout, &txResult)
// 	require.NoError(t, err)

// 	// Extract proposal_id from submit_proposal event
// 	var proposalID string
// 	events, ok := txResult["events"].([]interface{})
// 	require.True(t, ok, "events should exist")
// 	for _, event := range events {
// 		eventMap := event.(map[string]interface{})
// 		if eventMap["type"].(string) == "submit_proposal" {
// 			attributes := eventMap["attributes"].([]interface{})
// 			for _, attr := range attributes {
// 				attrMap := attr.(map[string]interface{})
// 				if attrMap["key"].(string) == "proposal_id" {
// 					proposalID = attrMap["value"].(string)
// 					break
// 				}
// 			}
// 			break
// 		}
// 	}
// 	require.NotEmpty(t, proposalID, "proposal ID should be extracted from transaction")
// 	t.Logf("Extracted proposal ID: %s", proposalID)

// 	// Wait and check that proposal exists and is in voting period
// 	t.Log("⏳ Waiting for proposal to enter voting period...")
// 	var proposalFound bool
// 	for i := 0; i < 10; i++ {
// 		queryCmd := []string{chain.Config().Bin, "query", "gov", "proposal", proposalID, "--node", chain.GetRPCAddress(), "--output", "json"}
// 		stdout, _, err = node.Exec(ctx, queryCmd, nil)
// 		t.Logf("Query proposal attempt %d: error=%v, stdout=%s", i+1, err, string(stdout))
// 		if err == nil {
// 			var proposalResult map[string]interface{}
// 			err = json.Unmarshal(stdout, &proposalResult)
// 			if err != nil {
// 				t.Logf("Failed to unmarshal proposal response: %v", err)
// 			} else if proposal, ok := proposalResult["proposal"].(map[string]interface{}); ok {
// 				status, _ := proposal["status"].(string)
// 				t.Logf("Proposal status: %s", status)
// 				if status == "PROPOSAL_STATUS_VOTING_PERIOD" {
// 					t.Logf("Proposal %s is in voting period", proposalID)
// 					proposalFound = true
// 					break
// 				}
// 			} else {
// 				t.Log("Proposal is nil in response")
// 			}
// 		}
// 		if i < 9 {
// 			time.Sleep(2 * time.Second)
// 		}
// 	}
// 	require.True(t, proposalFound, "proposal should exist and be in voting period")

// 	t.Log("🗳️  Voting on proposal...")
// 	// Vote on the proposal
// 	voteCmd := []string{
// 		chain.Config().Bin,
// 		"tx", "gov", "vote", proposalID, "yes",
// 		"--from", user.KeyName(),
// 		"--chain-id", chain.Config().ChainID,
// 		"--node", chain.GetRPCAddress(),
// 		"--home", node.HomeDir(),
// 		"--keyring-backend", "test",
// 		"--gas-prices", gasPrice,
// 		"--yes",
// 		"--output", "json",
// 	}

// 	stdout, _, err = node.Exec(ctx, voteCmd, nil)
// 	require.NoError(t, err)

// 	err = testutil.WaitForBlocks(ctx, 20, chain)
// 	require.NoError(t, err)

// 	// Query proposal status to verify it passed
// 	queryCmd := []string{chain.Config().Bin, "query", "gov", "proposal", proposalID, "--node", chain.GetRPCAddress(), "--output", "json"}
// 	stdout, _, err = node.Exec(ctx, queryCmd, nil)
// 	require.NoError(t, err)
// 	t.Logf("Full proposal query response: %s", string(stdout))

// 	var proposalResult map[string]interface{}
// 	err = json.Unmarshal(stdout, &proposalResult)
// 	require.NoError(t, err)
// 	proposal, ok := proposalResult["proposal"].(map[string]interface{})
// 	require.True(t, ok, "proposal field should exist")
// 	status, _ := proposal["status"].(string)
// 	require.Equal(t, "PROPOSAL_STATUS_PASSED", status)

// 	t.Log("⏳ Waiting for upgrade height...")
// 	currentHeight, err := chain.Height(ctx)
// 	require.NoError(t, err)

// 	if currentHeight < upgradeHeight {
// 		blocksToWait := upgradeHeight - currentHeight
// 		err = testutil.WaitForBlocks(ctx, int(blocksToWait), chain)
// 		require.NoError(t, err)
// 	}

// 	time.Sleep(10 * time.Second)

// 	t.Log("🔄 Performing chain upgrade...")
// 	chain.UpgradeVersion(ctx, client, postUpgradeVersion, ChainImage.Repository)

// 	err = testutil.WaitForBlocks(ctx, int(blocksAfterUpgrade), chain)
// 	require.NoError(t, err)

// 	// Post-upgrade validation
// 	t.Log("✅ Running post-upgrade validation...")

// 	postUpgradeHeight, err := chain.Height(ctx)
// 	require.NoError(t, err)
// 	require.Greater(t, postUpgradeHeight, upgradeHeight)
// 	t.Logf("Post-upgrade height: %d", postUpgradeHeight)

// 	// Verify state persistence
// 	recipientPostBalance, err := chain.GetBalance(ctx, recipient, Denom)
// 	require.NoError(t, err)
// 	require.Equal(t, recipientPreBalance, recipientPostBalance, "recipient balance should persist through upgrade")
// 	t.Logf("Post-upgrade recipient balance: %s%s (maintained)", recipientPostBalance.String(), Denom)

// 	// Verify chain functionality after upgrade
// 	postUpgradeTransferAmount := math.NewInt(3_000)
// 	err = chain.SendFunds(ctx, user.KeyName(), ibc.WalletAmount{
// 		Address: recipient,
// 		Denom:   Denom,
// 		Amount:  postUpgradeTransferAmount,
// 	})
// 	require.NoError(t, err)

// 	finalBalance, err := chain.GetBalance(ctx, recipient, Denom)
// 	require.NoError(t, err)
// 	expectedBalance := recipientPostBalance.Add(postUpgradeTransferAmount)
// 	require.True(t, finalBalance.GTE(expectedBalance), "post-upgrade transfer should work")
// 	t.Logf("Final recipient balance after post-upgrade transfer: %s%s", finalBalance.String(), Denom)

// 	// Verify upgrade was applied
// 	stdout, _, err = node.ExecQuery(ctx, "upgrade", "applied", upgradeName)
// 	require.NoError(t, err)
// 	require.NotEmpty(t, stdout)
// 	t.Logf("Upgrade %s successfully applied", upgradeName)

// 	// Additional validation: query module parameters
// 	stdout, _, err = node.ExecQuery(ctx, "bank", "total")
// 	require.NoError(t, err)
// 	require.NotEmpty(t, stdout)

// 	t.Log("✅ All post-upgrade validations passed successfully")
// }
