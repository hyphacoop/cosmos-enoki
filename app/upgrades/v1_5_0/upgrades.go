package v1_5_0

import (
	"context"

	"github.com/hyphacoop/cosmos-enoki/app/upgrades"

	errorsmod "cosmossdk.io/errors"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const UpgradeName = "v1.5.0"

// NewUpgrade constructor
func NewUpgrade() upgrades.Upgrade {
	return upgrades.Upgrade{
		UpgradeName:          UpgradeName,
		CreateUpgradeHandler: CreateUpgradeHandler,
	}
}

func CreateUpgradeHandler(
	mm upgrades.ModuleManager,
	configurator module.Configurator,
	ak *upgrades.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(c context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)

		ctx.Logger().Info("Starting v1.5.0 upgrade")
		// Add tokenfactory module to the version map since it's being added in this upgrade
		// fromVM[tokenfactorytypes.ModuleName] = 1

		// Run migrations to ensure compatibility with new modules
		fromVM, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return fromVM, errorsmod.Wrapf(err, "running module migrations")
		}

		ctx.Logger().Info("Upgrade v1.5.0 complete")
		return fromVM, nil

	}
}
