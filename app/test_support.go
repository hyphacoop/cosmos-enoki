package app

import (
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"

	"github.com/cosmos/cosmos-sdk/baseapp"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
)

func (app *EnokiApp) GetIBCKeeper() *ibckeeper.Keeper {
	return app.IBCKeeper
}

// func (app *ChainApp) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
// 	return app.ScopedIBCKeeper
// }

func (app *EnokiApp) GetBaseApp() *baseapp.BaseApp {
	return app.BaseApp
}

func (app *EnokiApp) GetBankKeeper() bankkeeper.Keeper {
	return app.BankKeeper
}

func (app *EnokiApp) GetStakingKeeper() *stakingkeeper.Keeper {
	return app.StakingKeeper
}

func (app *EnokiApp) GetAccountKeeper() authkeeper.AccountKeeper {
	return app.AccountKeeper
}

func (app *EnokiApp) GetWasmKeeper() wasmkeeper.Keeper {
	return app.WasmKeeper
}
