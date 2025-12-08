package app

import (
	"fmt"

	"github.com/hyphacoop/cosmos-enoki/app/upgrades"
	"github.com/hyphacoop/cosmos-enoki/app/upgrades/noop"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	v1_5_0 "github.com/hyphacoop/cosmos-enoki/app/upgrades/v1_5_0"
	v1_6_0 "github.com/hyphacoop/cosmos-enoki/app/upgrades/v1_6_0"
)

// Upgrades list of chain upgrades
var Upgrades = []upgrades.Upgrade{
	v1_5_0.NewUpgrade(),
	v1_6_0.NewUpgrade(),
}

// RegisterUpgradeHandlers registers the chain upgrade handlers
func (app *EnokiApp) RegisterUpgradeHandlers() {
	// setupLegacyKeyTables(&app.ParamsKeeper)
	if len(Upgrades) == 0 {
		// always have a unique upgrade registered for the current version to test in system tests
		Upgrades = append(Upgrades, noop.NewUpgrade(app.Version()))
	}

	keepers := upgrades.AppKeepers{
		AccountKeeper:         &app.AccountKeeper,
		ParamsKeeper:          &app.ParamsKeeper,
		ConsensusParamsKeeper: &app.ConsensusParamsKeeper,
		IBCKeeper:             app.IBCKeeper,
		Codec:                 app.appCodec,
		GetStoreKey:           app.GetKey,
		TokenFactoryKeeper:    &app.TokenFactoryKeeper,
	}

	// register all upgrade handlers
	for _, upgrade := range Upgrades {
		app.UpgradeKeeper.SetUpgradeHandler(
			upgrade.UpgradeName,
			upgrade.CreateUpgradeHandler(
				app.ModuleManager,
				app.configurator,
				&keepers,
			),
		)
	}

	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Sprintf("failed to read upgrade info from disk %s", err))
	}

	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		return
	}

	// register store loader for current upgrade
	for _, upgrade := range Upgrades {
		if upgradeInfo.Name == upgrade.UpgradeName {
			app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &upgrade.StoreUpgrades)) // nolint:gosec
			break
		}
	}
}
