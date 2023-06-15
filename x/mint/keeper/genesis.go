package keeper

import (
	"github.com/celestiaorg/celestia-app/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the x/mint store with data from the genesis state.
func (keeper Keeper) InitGenesis(ctx sdk.Context, ak types.AccountKeeper, data *types.GenesisState) {
	keeper.SetMinter(ctx, data.Minter)
	// override the genesis time with the actual genesis time supplied in `InitChain`
	blockTime := ctx.BlockTime()
	gt := types.GenesisTime{
		GenesisTime: &blockTime,
	}
	keeper.SetGenesisTime(ctx, gt)
	ak.GetModuleAccount(ctx, types.ModuleName)
}

// ExportGenesis returns a x/mint GenesisState for the given context.
func (keeper Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	minter := keeper.GetMinter(ctx)
	return types.NewGenesisState(minter)
}