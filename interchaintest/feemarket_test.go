package e2e

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testreporter"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/zap/zaptest"
)

func TestFeemarket(t *testing.T) {
	ctx := context.Background()
	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)
	client, network := interchaintest.DockerSetup(t)

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		&DefaultChainSpec,
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	chain := chains[0].(*cosmos.CosmosChain)

	// Setup Interchain
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

	users := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), GenesisFundsAmount, chain)
	user := users[0]
	CheckBaseGasPrice(t, ctx, chain, user)
	CheckBasePriceIncrease(t, ctx, chain, user)
}

func CheckBaseGasPrice(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet) {
	transferAmount := "1000"
	baseGasPrice, err := QueryJSON(chain, ctx, "base_gas_price", "feemarket", "state")
	require.NoError(t, err)
	fmt.Println("State> Base gas price: ", baseGasPrice)
	require.NoError(t, err)
	baseGasPriceFloat, err := strconv.ParseFloat(baseGasPrice.String(), 64)
	lowGasPriceFloat := baseGasPriceFloat / 2

	newWallet, err := chain.BuildWallet(ctx, "recipient", "")
	require.NoError(t, err)

	// Submit a transaction with less than the base gas price
	command := chain.GetNode().TxCommand(user.KeyName(), "bank", "send", user.FormattedAddress(), newWallet.FormattedAddress(), transferAmount+Denom, "--gas-prices", fmt.Sprintf("%f", lowGasPriceFloat)+Denom)
	so, _, err := chain.GetNode().Exec(ctx, command, nil)
	require.NoError(t, err)
	code := gjson.Get(string(so), "code")
	require.Equal(t, "13", code.String())

	newBalance, err := QueryJSON(chain, ctx, "balances", "bank", "balances", newWallet.FormattedAddress())
	require.NoError(t, err)
	fmt.Println("New balance: ", newBalance.String())
	require.Equal(t, "[]", newBalance.String())

	// Submit a transaction with the gas price set to the base gas price
	command = chain.GetNode().TxCommand(user.KeyName(), "bank", "send", user.FormattedAddress(), newWallet.FormattedAddress(), "1000uoki", "--gas-prices", baseGasPrice.Str+"uoki")
	output, _, err := chain.GetNode().Exec(ctx, command, nil)
	fmt.Println("Output: ", string(output))
	time.Sleep(5 * time.Second)
	newBalance, err = QueryJSON(chain, ctx, "balances.0.amount", "bank", "balances", newWallet.FormattedAddress())
	fmt.Println("Balance: ", newBalance.String())
	require.Equal(t, "1000", newBalance.String())
}

func CheckBasePriceIncrease(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet) {
	// Submit large proposal
	minDeposit, err := QueryJSON(chain, ctx, "params.min_deposit.0.amount", "gov", "params")
	require.NoError(t, err)
	fmt.Println("Min deposit: ", minDeposit)

	maxGasInt, err := strconv.Atoi(FeeMarketMaxGas)
	require.NoError(t, err)
	summarySize := int(maxGasInt / 400)
	fmt.Println("Seting a payload", summarySize, "bytes in size.")

	proposal, err := chain.BuildProposal(nil, "Test Proposal", "placeholder", "ipfs://CID", minDeposit.String()+Denom, user.KeyName(), false)
	require.NoError(t, err)
	payload, err := RandomHexString(summarySize)
	require.NoError(t, err)
	proposal.Summary = payload

	baseGasPrice, err := QueryJSON(chain, ctx, "base_gas_price", "feemarket", "state")
	require.NoError(t, err)
	fmt.Println("State> Base gas price before proposal: ", baseGasPrice)
	result, err := chain.SubmitProposal(ctx, user.KeyName(), proposal)
	require.NoError(t, err)
	proposalHeight := result.Height

	gasUsed, err := QueryJSON(chain, ctx, "gas_used", "tx", result.TxHash)
	require.NoError(t, err)
	fmt.Println("Gas used: ", gasUsed.Int())

	testGasPrice, err := QueryJSON(chain, ctx, "base_gas_price", "feemarket", "state", "--height", strconv.Itoa(int(proposalHeight)))
	require.NoError(t, err)
	fmt.Println("State> Base gas price at proposal height: ", testGasPrice)

	startAmount, err := strconv.ParseFloat(baseGasPrice.String(), 64)
	require.NoError(t, err)
	endAmount, err := strconv.ParseFloat(testGasPrice.String(), 64)
	require.NoError(t, err)
	require.Greater(t, endAmount, startAmount)
}

func RandomHexString(n int) (string, error) {
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
