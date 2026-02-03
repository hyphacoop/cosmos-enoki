package e2e

// Upgrade Helper Functions
//
// This file contains reusable helper functions for performing chain upgrades in interchaintest.
// These functions can be used across multiple test files (upgrade_test.go, ibc_test.go, etc.)
// to avoid code duplication.
//
// Usage Example:
//
//	// In any test file (e.g., ibc_test.go):
//	params := UpgradeParams{
//	    ChainName:          chain.Config().Name,
//	    UpgradeName:        "v1.7.0",
//	    UpgradeHeight:      50,
//	    PostUpgradeVersion: "v1.7.0",
//	}
//	PerformUpgrade(ctx, t, chain, client, userKeyName, params)
//
// Available Functions:
//   - GetGovMinDeposit: Query governance minimum deposit amount
//   - GetFeeMarketGasPrice: Query current gas price from feemarket
//   - SubmitUpgradeProposal: Submit a software upgrade proposal and return proposal ID
//   - WaitForProposalVotingPeriod: Wait for proposal to enter voting period
//   - VoteOnProposal: Vote yes on a proposal using validator key
//   - WaitForProposalPass: Wait for proposal to pass and verify status
//   - ExecuteUpgrade: Perform the chain upgrade at specified height
//   - VerifyUpgradeApplied: Verify upgrade was successfully applied
//   - VerifyChainFunctionality: Test basic chain operations post-upgrade
//   - PerformUpgrade: Complete upgrade workflow (all steps above)

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	client "github.com/moby/moby/client"
	"github.com/stretchr/testify/require"
)

const (
	// GovModuleAuthority is the governance module authority address
	GovModuleAuthority = "enoki10d07y265gmmuvt4z0w9aw880jnsr700jkqmfe9"
)

// UpgradeParams contains parameters needed for a chain upgrade
type UpgradeParams struct {
	UpgradeName        string
	UpgradeHeight      int64
	PostUpgradeVersion string
	SkipProposal       bool
}

// GetGovMinDeposit queries the chain for the minimum governance proposal deposit
func GetGovMinDeposit(ctx context.Context, t *testing.T, chain *cosmos.CosmosChain) string {
	t.Helper()

	node := chain.GetNode()
	govParamsCmd := []string{chain.Config().Bin, "query", "gov", "params", "--node", chain.GetRPCAddress(), "--output", "json"}
	stdout, _, err := node.Exec(ctx, govParamsCmd, nil)
	require.NoError(t, err)

	var paramsResult map[string]interface{}
	err = json.Unmarshal(stdout, &paramsResult)
	require.NoError(t, err)

	params, ok := paramsResult["params"].(map[string]interface{})
	require.True(t, ok, "params field should exist")

	minDepositArray, ok := params["min_deposit"].([]interface{})
	require.True(t, ok && len(minDepositArray) > 0, "min_deposit should be a non-empty array")

	minDepositCoin := minDepositArray[0].(map[string]interface{})
	depositAmount := fmt.Sprintf("%s%s", minDepositCoin["amount"].(string), minDepositCoin["denom"].(string))

	t.Logf("Minimum deposit required: %s", depositAmount)
	return depositAmount
}

// GetFeeMarketGasPrice queries the chain for the current gas price from feemarket module
func GetFeeMarketGasPrice(ctx context.Context, t *testing.T, chain *cosmos.CosmosChain) string {
	t.Helper()

	node := chain.GetNode()
	gasPricesCmd := []string{chain.Config().Bin, "query", "feemarket", "gas-prices", "--node", chain.GetRPCAddress(), "--output", "json"}
	stdout, _, err := node.Exec(ctx, gasPricesCmd, nil)
	require.NoError(t, err)

	var gasPricesResult map[string]interface{}
	err = json.Unmarshal(stdout, &gasPricesResult)
	require.NoError(t, err)

	pricesArray, ok := gasPricesResult["prices"].([]interface{})
	require.True(t, ok && len(pricesArray) > 0, "prices should be a non-empty array")

	priceObj := pricesArray[0].(map[string]interface{})
	gasPrice := fmt.Sprintf("%s%s", priceObj["amount"].(string), priceObj["denom"].(string))

	t.Logf("Gas price: %s", gasPrice)
	return gasPrice
}

// SubmitUpgradeProposal submits a software upgrade proposal and returns the proposal ID
func SubmitUpgradeProposal(ctx context.Context, t *testing.T, chain *cosmos.CosmosChain, userKeyName string, params UpgradeParams) string {
	t.Helper()

	node := chain.GetNode()
	depositAmount := GetGovMinDeposit(ctx, t, chain)
	gasPrice := GetFeeMarketGasPrice(ctx, t, chain)

	// Create the proposal content
	proposalJSON := fmt.Sprintf(`{
		"messages": [{
			"@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
			"authority": "%s",
			"plan": {
				"name": "%s",
				"height": "%d",
				"info": "Upgrade to %s"
			}
		}],
		"metadata": "ipfs://CID",
		"deposit": "%s",
		"title": "Upgrade to %s",
		"summary": "This proposal will upgrade the chain to %s at height %d"
	}`, GovModuleAuthority, params.UpgradeName, params.UpgradeHeight, params.UpgradeName, depositAmount, params.UpgradeName, params.UpgradeName, params.UpgradeHeight)

	// Write proposal JSON to file in node's home directory
	proposalFile := node.HomeDir() + "/upgrade-proposal.json"
	writeCmd := []string{"sh", "-c", fmt.Sprintf("echo '%s' > %s", proposalJSON, proposalFile)}
	_, _, err := node.Exec(ctx, writeCmd, nil)
	require.NoError(t, err)

	// Submit proposal
	cmd := []string{
		chain.Config().Bin,
		"tx", "gov", "submit-proposal", proposalFile,
		"--from", userKeyName,
		"--chain-id", chain.Config().ChainID,
		"--node", chain.GetRPCAddress(),
		"--home", node.HomeDir(),
		"--keyring-backend", "test",
		"--gas", "auto",
		"--gas-adjustment", "2.0",
		"--gas-prices", gasPrice,
		"--yes",
		"--output", "json",
	}

	stdout, _, err := node.Exec(ctx, cmd, nil)
	require.NoError(t, err)
	t.Logf("Proposal submission output: %s", string(stdout))

	// Wait for proposal to be processed
	err = testutil.WaitForBlocks(ctx, 2, chain)
	require.NoError(t, err)

	// Extract transaction hash from submission output
	var txResponse map[string]interface{}
	err = json.Unmarshal(stdout, &txResponse)
	require.NoError(t, err)
	txHash, ok := txResponse["txhash"].(string)
	require.True(t, ok && txHash != "", "transaction hash should be present")
	t.Logf("Transaction hash: %s", txHash)

	// Query transaction to extract proposal ID from events
	txQueryCmd := []string{chain.Config().Bin, "query", "tx", txHash, "--node", chain.GetRPCAddress(), "--output", "json"}
	stdout, _, err = node.Exec(ctx, txQueryCmd, nil)
	require.NoError(t, err)

	var txResult map[string]interface{}
	err = json.Unmarshal(stdout, &txResult)
	require.NoError(t, err)

	// Extract proposal_id from submit_proposal event
	var proposalID string
	events, ok := txResult["events"].([]interface{})
	require.True(t, ok, "events should exist")
	for _, event := range events {
		eventMap := event.(map[string]interface{})
		if eventMap["type"].(string) == "submit_proposal" {
			attributes := eventMap["attributes"].([]interface{})
			for _, attr := range attributes {
				attrMap := attr.(map[string]interface{})
				if attrMap["key"].(string) == "proposal_id" {
					proposalID = attrMap["value"].(string)
					break
				}
			}
			break
		}
	}
	require.NotEmpty(t, proposalID, "proposal ID should be extracted from transaction")
	t.Logf("Extracted proposal ID: %s", proposalID)

	return proposalID
}

// WaitForProposalVotingPeriod waits for a proposal to enter voting period
func WaitForProposalVotingPeriod(ctx context.Context, t *testing.T, chain *cosmos.CosmosChain, proposalID string) {
	t.Helper()

	node := chain.GetNode()
	t.Log("⏳ Waiting for proposal to enter voting period...")

	var proposalFound bool
	for i := 0; i < 10; i++ {
		queryCmd := []string{chain.Config().Bin, "query", "gov", "proposal", proposalID, "--node", chain.GetRPCAddress(), "--output", "json"}
		stdout, _, err := node.Exec(ctx, queryCmd, nil)
		t.Logf("Query proposal attempt %d: error=%v, stdout=%s", i+1, err, string(stdout))

		if err == nil {
			var proposalResult map[string]interface{}
			err = json.Unmarshal(stdout, &proposalResult)
			if err != nil {
				t.Logf("Failed to unmarshal proposal response: %v", err)
			} else if proposal, ok := proposalResult["proposal"].(map[string]interface{}); ok {
				status, _ := proposal["status"].(string)
				t.Logf("Proposal status: %s", status)
				if status == "PROPOSAL_STATUS_VOTING_PERIOD" {
					t.Logf("Proposal %s is in voting period", proposalID)
					proposalFound = true
					break
				}
			} else {
				t.Log("Proposal is nil in response")
			}
		}

		if i < 9 {
			time.Sleep(2 * time.Second)
		}
	}

	require.True(t, proposalFound, "proposal should exist and be in voting period")
}

// VoteOnProposal votes yes on a proposal using the validator key
func VoteOnProposal(ctx context.Context, t *testing.T, chain *cosmos.CosmosChain, proposalID string, voterKeyName string) {
	t.Helper()

	validators := chain.Validators
	require.NotEmpty(t, validators, "chain should have validators")
	validator := validators[0]

	gasPrice := GetFeeMarketGasPrice(ctx, t, chain)

	voteCmd := []string{
		chain.Config().Bin,
		"tx", "gov", "vote", proposalID, "yes",
		"--from", voterKeyName,
		"--chain-id", chain.Config().ChainID,
		"--node", chain.GetRPCAddress(),
		"--home", validator.HomeDir(),
		"--keyring-backend", "test",
		"--gas-prices", gasPrice,
		"--yes",
		"--output", "json",
	}

	stdout, _, err := validator.Exec(ctx, voteCmd, nil)
	require.NoError(t, err)
	t.Logf("Vote output: %s", string(stdout))
}

// WaitForProposalPass waits for a proposal to pass and verifies its status
func WaitForProposalPass(ctx context.Context, t *testing.T, chain *cosmos.CosmosChain, proposalID string) {
	t.Helper()

	// Wait for voting period to end
	err := testutil.WaitForBlocks(ctx, 20, chain)
	require.NoError(t, err)

	// Query proposal status to verify it passed
	node := chain.GetNode()
	queryCmd := []string{chain.Config().Bin, "query", "gov", "proposal", proposalID, "--node", chain.GetRPCAddress(), "--output", "json"}
	stdout, _, err := node.Exec(ctx, queryCmd, nil)
	require.NoError(t, err)
	t.Logf("Full proposal query response: %s", string(stdout))

	var proposalResult map[string]interface{}
	err = json.Unmarshal(stdout, &proposalResult)
	require.NoError(t, err)

	proposal, ok := proposalResult["proposal"].(map[string]interface{})
	require.True(t, ok, "proposal field should exist")

	status, _ := proposal["status"].(string)
	require.Equal(t, "PROPOSAL_STATUS_PASSED", status, "proposal should have passed")
	t.Logf("✅ Proposal %s passed", proposalID)
}

// ExecuteUpgrade performs the chain upgrade at the specified height
func ExecuteUpgrade(ctx context.Context, t *testing.T, chain *cosmos.CosmosChain, client *client.Client, params UpgradeParams) {
	t.Helper()

	// Wait for chain to reach upgrade height
	currentHeight, err := chain.Height(ctx)
	require.NoError(t, err)

	if currentHeight < params.UpgradeHeight {
		blocksToWait := params.UpgradeHeight - currentHeight
		t.Logf("⏳ Waiting %d blocks to reach upgrade height %d", blocksToWait, params.UpgradeHeight)
		err = testutil.WaitForBlocks(ctx, int(blocksToWait), chain)
		require.NoError(t, err)
	}

	// Wait a bit for the upgrade to be triggered
	time.Sleep(10 * time.Second)

	// Stop nodes to prepare for upgrade
	t.Log("Stopping nodes for upgrade...")
	err = chain.StopAllNodes(ctx)
	require.NoError(t, err)

	// Upgrade the chain binary using the pre-loaded image repository
	t.Logf("🔄 Upgrading chain to version %s", params.PostUpgradeVersion)
	chain.UpgradeVersion(ctx, client, chain.GetNode().Image.Repository, params.PostUpgradeVersion)

	// Start all nodes back up with new version
	t.Log("Starting nodes with new version...")
	err = chain.StartAllNodes(ctx)
	require.NoError(t, err)

	// Wait for chain to produce blocks after upgrade
	err = testutil.WaitForBlocks(ctx, 10, chain)
	require.NoError(t, err)

	// Verify chain is running post-upgrade
	height, err := chain.Height(ctx)
	require.NoError(t, err)
	require.Greater(t, height, params.UpgradeHeight, "chain should be running after upgrade")
	t.Logf("✅ Chain is running at height %d after upgrade", height)
}

// VerifyUpgradeApplied verifies that the upgrade was successfully applied
func VerifyUpgradeApplied(ctx context.Context, t *testing.T, chain *cosmos.CosmosChain, upgradeName string) {
	t.Helper()

	node := chain.GetNode()
	stdout, _, err := node.ExecQuery(ctx, "upgrade", "applied", upgradeName)
	require.NoError(t, err)
	require.NotEmpty(t, stdout)
	t.Logf("✅ Upgrade %s was successfully applied", upgradeName)
}

// VerifyChainFunctionality verifies basic chain functionality after upgrade
func VerifyChainFunctionality(ctx context.Context, t *testing.T, chain *cosmos.CosmosChain, userKeyName string) {
	t.Helper()

	recipient := "enoki1hj5fveer5cjtn4wd6wstzugjfdxzl0xp2w67r4"
	transferAmount := math.NewInt(1_000)

	err := chain.SendFunds(ctx, userKeyName, ibc.WalletAmount{
		Address: recipient,
		Denom:   Denom,
		Amount:  transferAmount,
	})
	require.NoError(t, err)

	// Verify the transfer was successful
	balance, err := chain.GetBalance(ctx, recipient, Denom)
	require.NoError(t, err)
	require.True(t, balance.GTE(transferAmount), "recipient should have received funds")
	t.Log("✅ Basic chain functionality verified (bank transfer successful)")
}

// PerformUpgrade orchestrates a complete upgrade workflow
func PerformUpgrade(ctx context.Context, t *testing.T, chain *cosmos.CosmosChain, client *client.Client, userKeyName string, params UpgradeParams) {
	t.Helper()

	// Submit and vote on upgrade proposal
	proposalID := SubmitUpgradeProposal(ctx, t, chain, userKeyName, params)
	WaitForProposalVotingPeriod(ctx, t, chain, proposalID)
	VoteOnProposal(ctx, t, chain, proposalID, "validator")
	WaitForProposalPass(ctx, t, chain, proposalID)

	// Execute the upgrade
	ExecuteUpgrade(ctx, t, chain, client, params)

	// Verify upgrade was applied
	VerifyUpgradeApplied(ctx, t, chain, params.UpgradeName)

	// Verify chain functionality
	VerifyChainFunctionality(ctx, t, chain, userKeyName)

	t.Logf("✅ Complete upgrade workflow finished successfully")
}
