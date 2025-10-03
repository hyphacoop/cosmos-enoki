package v1_4_0

import (
	"context"

	"github.com/hyphacoop/cosmos-enoki/app/upgrades"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	tokenfactorytypes "github.com/strangelove-ventures/tokenfactory/x/tokenfactory/types"
)

const UpgradeName = "v1.4.0"

// initializeTokenFactoryParams sets up the initial parameters for the tokenfactory module
func initializeTokenFactoryParams(ctx sdk.Context, ak *upgrades.AppKeepers) error {
	ctx.Logger().Info("Initializing tokenfactory parameters")
	// Set tokenfactory parameters after module initialization
	// Set denom creation fee to 100_000_000uoki
	denomCreationFee := sdk.NewCoins(sdk.NewInt64Coin("uoki", 100_000_000))

	// Set denom creation gas consume to 100,000
	denomCreationGasConsume := uint64(100000)

	// Create the new parameters
	params := tokenfactorytypes.Params{
		DenomCreationFee:        denomCreationFee,
		DenomCreationGasConsume: denomCreationGasConsume,
	}

	// Set the parameters in the tokenfactory keeper
	return ak.TokenFactoryKeeper.SetParams(ctx, params)
}

// NewUpgrade constructor
func NewUpgrade() upgrades.Upgrade {
	return upgrades.Upgrade{
		UpgradeName:          UpgradeName,
		CreateUpgradeHandler: CreateUpgradeHandler,
		StoreUpgrades: storetypes.StoreUpgrades{
			Added:   []string{tokenfactorytypes.StoreKey},
			Deleted: []string{},
		},
	}
}

func CreateUpgradeHandler(
	mm upgrades.ModuleManager,
	configurator module.Configurator,
	ak *upgrades.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(c context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)

		ctx.Logger().Info("Starting v1.4.0 upgrade")
		// Add tokenfactory module to the version map since it's being added in this upgrade
		// fromVM[tokenfactorytypes.ModuleName] = 1

		// Run migrations to ensure compatibility with new modules
		fromVM, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return fromVM, errorsmod.Wrapf(err, "running module migrations")
		}

		// Initialize tokenfactory parameters
		if err := initializeTokenFactoryParams(ctx, ak); err != nil {
			return nil, errorsmod.Wrapf(err, "initializing tokenfactory parameters")
		}

		ctx.Logger().Info("Upgrade v1.4.0 complete")
		return fromVM, nil

	}
}
