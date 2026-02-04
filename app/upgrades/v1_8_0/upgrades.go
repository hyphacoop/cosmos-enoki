package v1_8_0

import (
	"context"

	"github.com/hyphacoop/cosmos-enoki/app/upgrades"

	errorsmod "cosmossdk.io/errors"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const UpgradeName = "v1.8.0"

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

		ctx.Logger().Info("Starting upgrade", "name", UpgradeName)

		// Run migrations to ensure compatibility with new modules
		fromVM, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return fromVM, errorsmod.Wrapf(err, "running module migrations")
		}

		ctx.Logger().Info("Upgrade complete", "name", UpgradeName)
		return fromVM, nil

	}
}
