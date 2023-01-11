package hard_borrow

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kava-labs/kava/x/incentive/types"
)

var _ types.SourceAdapter = SourceAdapter{}

type SourceAdapter struct {
	keeper types.HardKeeper
}

func NewSourceAdapter(keeper types.HardKeeper) SourceAdapter {
	return SourceAdapter{
		keeper: keeper,
	}
}

func (f SourceAdapter) TotalSharesBySource(ctx sdk.Context, sourceID string) sdk.Dec {
	coins, found := f.keeper.GetBorrowedCoins(ctx)
	if !found {
		return sdk.ZeroDec()
	}

	totalBorrowed := coins.AmountOf(sourceID).ToDec()

	interestFactor, found := f.keeper.GetBorrowInterestFactor(ctx, sourceID)
	if !found {
		// assume nothing has been borrowed so the factor starts at it's default value
		interestFactor = sdk.OneDec()
	}

	// return borrowed/factor to get the "pre interest" value of the current total borrowed
	return totalBorrowed.Quo(interestFactor)
}

func (f SourceAdapter) OwnerSharesBySource(
	ctx sdk.Context,
	owner sdk.AccAddress,
	sourceIDs []string,
) map[string]sdk.Dec {
	borrowCoins := sdk.NewDecCoins()

	accBorrow, found := f.keeper.GetBorrow(ctx, owner)
	if found {
		normalizedBorrow, err := accBorrow.NormalizedBorrow()
		if err != nil {
			panic(fmt.Errorf("failed to normalize hard borrow for owner %s: %w", owner, err))
		}

		borrowCoins = normalizedBorrow
	}

	shares := make(map[string]sdk.Dec)
	for _, id := range sourceIDs {
		shares[id] = borrowCoins.AmountOf(id)
	}

	return shares
}
