package v2_0_0

import (
	"context"

	"github.com/hyphacoop/cosmos-enoki/app/upgrades"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	grouptypes "github.com/cosmos/cosmos-sdk/x/group"
)

const UpgradeName = "v2.0.0"

// NewUpgrade constructor
func NewUpgrade() upgrades.Upgrade {
	return upgrades.Upgrade{
		UpgradeName:          UpgradeName,
		CreateUpgradeHandler: CreateUpgradeHandler,
		StoreUpgrades: storetypes.StoreUpgrades{
			Deleted: []string{grouptypes.ModuleName},
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

		ctx.Logger().Info("Starting upgrade", "name", UpgradeName)

		// Remove x/group from the version map so it is excluded from migrations
		ctx.Logger().Info("Removing x/group module")
		delete(fromVM, grouptypes.ModuleName)

		// Run migrations
		fromVM, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return fromVM, errorsmod.Wrapf(err, "running module migrations")
		}

		ctx.Logger().Info("Upgrade complete", "name", UpgradeName)
		return fromVM, nil
	}
}
