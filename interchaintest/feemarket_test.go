package e2e

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testreporter"
	"github.com/stretchr/testify/require"
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

	CheckBasePriceIncrease(t, ctx, chain, user)
}

func CheckBasePriceIncrease(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet) (contractAddr string) {
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

	return
}

func RandomHexString(n int) (string, error) {
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
