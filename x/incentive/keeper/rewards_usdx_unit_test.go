package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	cdptypes "github.com/kava-labs/kava/x/cdp/types"
	"github.com/kava-labs/kava/x/incentive/types"
)

// usdxRewardsUnitTester contains common methods for running unit tests for keeper methods related to the USDX minting rewards
type usdxRewardsUnitTester struct {
	unitTester
}

func (suite *usdxRewardsUnitTester) storeGlobalUSDXIndexes(indexes types.RewardIndexes) {
	for _, ri := range indexes {
		suite.keeper.SetUSDXMintingRewardFactor(suite.ctx, ri.CollateralType, ri.RewardFactor)
	}
}

func (suite *usdxRewardsUnitTester) storeClaim(claim types.Claim) {
	suite.keeper.SetClaim(suite.ctx, claim)
}

type InitializeUSDXMintingClaimTests struct {
	usdxRewardsUnitTester
}

func TestInitializeUSDXMintingClaims(t *testing.T) {
	suite.Run(t, new(InitializeUSDXMintingClaimTests))
}

func (suite *InitializeUSDXMintingClaimTests) TestClaimIndexIsSetWhenClaimDoesNotExist() {
	collateralType := "bnb-a"

	cdp := NewCDPBuilder(arbitraryAddress(), collateralType).Build()

	globalIndexes := types.RewardIndexes{{
		CollateralType: collateralType,
		RewardFactor:   d("0.2"),
	}}
	suite.storeGlobalUSDXIndexes(globalIndexes)

	suite.keeper.InitializeUSDXMintingClaim(suite.ctx, cdp)

	syncedClaim, f := suite.keeper.GetClaim(suite.ctx, types.CLAIM_TYPE_USDX_MINTING, cdp.Owner)
	suite.True(f)
	suite.Equal(usdxRewardIndexToMultiRewardIndexes(globalIndexes), syncedClaim.RewardIndexes)
}

func (suite *InitializeUSDXMintingClaimTests) TestClaimIndexIsSetWhenClaimExists() {
	collateralType := "bnb-a"

	claim := types.Claim{
		Type:  types.CLAIM_TYPE_USDX_MINTING,
		Owner: arbitraryAddress(),
		RewardIndexes: types.MultiRewardIndexes{
			{
				CollateralType: collateralType,
				RewardIndexes: types.RewardIndexes{{
					CollateralType: types.USDXMintingRewardDenom,
					RewardFactor:   d("0.1"),
				}},
			}},
	}
	suite.storeClaim(claim)

	globalIndexes := types.RewardIndexes{{
		CollateralType: collateralType,
		RewardFactor:   d("0.2"),
	}}
	suite.storeGlobalUSDXIndexes(globalIndexes)

	cdp := NewCDPBuilder(claim.Owner, collateralType).Build()

	suite.keeper.InitializeUSDXMintingClaim(suite.ctx, cdp)

	syncedClaim, found := suite.keeper.GetClaim(suite.ctx, types.CLAIM_TYPE_USDX_MINTING, cdp.Owner)
	suite.True(found)

	suite.Equal(usdxRewardIndexToMultiRewardIndexes(globalIndexes), syncedClaim.RewardIndexes)
}

type SynchronizeUSDXMintingRewardTests struct {
	usdxRewardsUnitTester
}

func TestSynchronizeUSDXMintingReward(t *testing.T) {
	suite.Run(t, new(SynchronizeUSDXMintingRewardTests))
}

func (suite *SynchronizeUSDXMintingRewardTests) TestRewardUnchangedWhenGlobalIndexesUnchanged() {
	unchangingRewardIndexes := nonEmptyRewardIndexes
	collateralType := extractFirstCollateralType(unchangingRewardIndexes)

	claim := types.Claim{
		Type:          types.CLAIM_TYPE_USDX_MINTING,
		Owner:         arbitraryAddress(),
		Reward:        cs(c(types.USDXMintingRewardDenom, 0)),
		RewardIndexes: usdxRewardIndexToMultiRewardIndexes(unchangingRewardIndexes),
	}
	suite.storeClaim(claim)

	suite.storeGlobalUSDXIndexes(unchangingRewardIndexes)

	cdp := NewCDPBuilder(claim.Owner, collateralType).WithSourceShares(1e12).Build()

	suite.keeper.SynchronizeUSDXMintingReward(suite.ctx, cdp)

	syncedClaim, found := suite.keeper.GetClaim(suite.ctx, types.CLAIM_TYPE_USDX_MINTING, claim.Owner)
	suite.True(found)

	suite.True(claim.Reward.IsEqual(syncedClaim.Reward))
}

func (suite *SynchronizeUSDXMintingRewardTests) TestRewardIsIncrementedWhenGlobalIndexIncreased() {
	collateralType := "bnb-a"

	claim := types.Claim{
		Type:   types.CLAIM_TYPE_USDX_MINTING,
		Owner:  arbitraryAddress(),
		Reward: cs(c(types.USDXMintingRewardDenom, 0)),
		RewardIndexes: types.MultiRewardIndexes{
			{
				CollateralType: collateralType,
				RewardIndexes: types.RewardIndexes{{
					CollateralType: types.USDXMintingRewardDenom,
					RewardFactor:   d("0.1"),
				}},
			},
		},
	}
	suite.storeClaim(claim)

	globalIndexes := types.RewardIndexes{
		{
			CollateralType: collateralType,
			RewardFactor:   d("0.2"),
		},
	}
	suite.storeGlobalUSDXIndexes(globalIndexes)

	cdp := NewCDPBuilder(claim.Owner, collateralType).WithSourceShares(1e12).Build()

	suite.keeper.SynchronizeUSDXMintingReward(suite.ctx, cdp)

	syncedClaim, found := suite.keeper.GetClaim(suite.ctx, types.CLAIM_TYPE_USDX_MINTING, claim.Owner)
	suite.True(found)

	// reward is ( new index - old index ) * cdp.TotalPrincipal
	suite.Equal(cs(c(types.USDXMintingRewardDenom, 1e11)), syncedClaim.Reward)
}

func (suite *SynchronizeUSDXMintingRewardTests) TestClaimIndexIsUpdatedWhenGlobalIndexIncreased() {
	claimsRewardIndexes := nonEmptyRewardIndexes
	collateralType := extractFirstCollateralType(claimsRewardIndexes)

	claim := types.Claim{
		Type:          types.CLAIM_TYPE_USDX_MINTING,
		Owner:         arbitraryAddress(),
		Reward:        cs(c(types.USDXMintingRewardDenom, 0)),
		RewardIndexes: usdxRewardIndexToMultiRewardIndexes(claimsRewardIndexes),
	}
	suite.storeClaim(claim)

	globalIndexes := increaseRewardFactors(claimsRewardIndexes)
	suite.storeGlobalUSDXIndexes(globalIndexes)

	cdp := NewCDPBuilder(claim.Owner, collateralType).Build()

	suite.keeper.SynchronizeUSDXMintingReward(suite.ctx, cdp)

	syncedClaim, found := suite.keeper.GetClaim(suite.ctx, types.CLAIM_TYPE_USDX_MINTING, claim.Owner)
	suite.True(found)

	// Only the claim's index for `collateralType` should have been changed
	i, found := globalIndexes.Get(collateralType)
	suite.True(found)

	expectedIndexes := claimsRewardIndexes.With(collateralType, i)
	suite.Equal(usdxRewardIndexToMultiRewardIndexes(expectedIndexes), syncedClaim.RewardIndexes)
}

func (suite *SynchronizeUSDXMintingRewardTests) TestClaimIndexIsUpdatedWhenNewRewardAddedAndClaimAlreadyExists() {
	claimsRewardIndexes := types.RewardIndexes{
		{
			CollateralType: "bnb-a",
			RewardFactor:   d("0.1"),
		},
		{
			CollateralType: "busd-b",
			RewardFactor:   d("0.4"),
		},
	}
	newRewardIndex := types.NewRewardIndex("xrp-a", d("0.0001"))

	claim := types.Claim{
		Type:          types.CLAIM_TYPE_USDX_MINTING,
		Owner:         arbitraryAddress(),
		Reward:        cs(c(types.USDXMintingRewardDenom, 0)),
		RewardIndexes: usdxRewardIndexToMultiRewardIndexes(claimsRewardIndexes),
	}
	suite.storeClaim(claim)

	globalIndexes := increaseRewardFactors(claimsRewardIndexes)
	globalIndexes = append(globalIndexes, newRewardIndex)
	suite.storeGlobalUSDXIndexes(globalIndexes)

	cdp := NewCDPBuilder(claim.Owner, newRewardIndex.CollateralType).Build()

	suite.keeper.SynchronizeUSDXMintingReward(suite.ctx, cdp)

	syncedClaim, found := suite.keeper.GetClaim(suite.ctx, types.CLAIM_TYPE_USDX_MINTING, claim.Owner)
	suite.True(found)

	// Only the claim's index for `collateralType` should have been changed
	expectedIndexes := claimsRewardIndexes.With(newRewardIndex.CollateralType, newRewardIndex.RewardFactor)
	suite.Equal(usdxRewardIndexToMultiRewardIndexes(expectedIndexes), syncedClaim.RewardIndexes)
}

func (suite *SynchronizeUSDXMintingRewardTests) TestClaimIsUnchangedWhenGlobalFactorMissing() {
	claimsRewardIndexes := nonEmptyRewardIndexes
	claim := types.Claim{
		Type:   types.CLAIM_TYPE_USDX_MINTING,
		Owner:  arbitraryAddress(),
		Reward: cs(c(types.USDXMintingRewardDenom, 0)),
		RewardIndexes: types.MultiRewardIndexes{
			{
				CollateralType: types.USDXMintingRewardDenom,
				RewardIndexes:  claimsRewardIndexes,
			},
		},
	}
	suite.storeClaim(claim)
	// don't store any reward indexes

	// create a cdp with collateral type that doesn't exist in the claim's indexes, and does not have a corresponding global factor
	cdp := NewCDPBuilder(claim.Owner, "unrewardedcollateral").WithSourceShares(1e12).Build()

	suite.keeper.SynchronizeUSDXMintingReward(suite.ctx, cdp)

	syncedClaim, found := suite.keeper.GetClaim(suite.ctx, types.CLAIM_TYPE_USDX_MINTING, claim.Owner)
	suite.True(found)
	suite.Equal(claim.RewardIndexes, syncedClaim.RewardIndexes)
	suite.True(claim.Reward.IsEqual(syncedClaim.Reward))
}

// CDPBuilder is a tool for creating a CDP in tests.
// The builder inherits from cdp.CDP, so fields can be accessed directly if a helper method doesn't exist.
type CDPBuilder struct {
	cdptypes.CDP
}

// NewCDPBuilder creates a CdpBuilder containing a CDP with owner and collateral type set.
func NewCDPBuilder(owner sdk.AccAddress, collateralType string) CDPBuilder {
	return CDPBuilder{
		CDP: cdptypes.CDP{
			Owner: owner,
			Type:  collateralType,
			// The zero value of Principal and AccumulatedFees (type sdk.Coin) is invalid as the denom is ""
			// Set them to the default denom, but with 0 amount.
			Principal:       c(cdptypes.DefaultStableDenom, 0),
			AccumulatedFees: c(cdptypes.DefaultStableDenom, 0),
			// zero value of sdk.Dec causes nil pointer panics
			InterestFactor: sdk.OneDec(),
		},
	}
}

// Build assembles and returns the final deposit.
func (builder CDPBuilder) Build() cdptypes.CDP { return builder.CDP }

// WithSourceShares adds a principal amount and interest factor such that the source shares for this CDP is equal to specified.
// With a factor of 1, the total principal is the source shares. This picks an arbitrary factor to ensure factors are accounted for in production code.
func (builder CDPBuilder) WithSourceShares(shares int64) CDPBuilder {
	if !builder.GetTotalPrincipal().Amount.Equal(sdk.ZeroInt()) {
		panic("setting source shares on cdp with existing principal or fees not implemented")
	}
	if !(builder.InterestFactor.IsNil() || builder.InterestFactor.Equal(sdk.OneDec())) {
		panic("setting source shares on cdp with existing interest factor not implemented")
	}
	// pick arbitrary interest factor
	factor := sdk.NewInt(2)

	// Calculate deposit amount that would equal the requested source shares given the above factor.
	principal := sdk.NewInt(shares).Mul(factor)

	builder.Principal = sdk.NewCoin(cdptypes.DefaultStableDenom, principal)
	builder.InterestFactor = factor.ToDec()

	return builder
}

func (builder CDPBuilder) WithPrincipal(principal sdk.Int) CDPBuilder {
	builder.Principal = sdk.NewCoin(cdptypes.DefaultStableDenom, principal)
	return builder
}

var nonEmptyRewardIndexes = types.RewardIndexes{
	{
		CollateralType: "bnb-a",
		RewardFactor:   d("0.1"),
	},
	{
		CollateralType: "busd-b",
		RewardFactor:   d("0.4"),
	},
}

func extractFirstCollateralType(indexes types.RewardIndexes) string {
	if len(indexes) == 0 {
		panic("cannot extract a collateral type from 0 length RewardIndexes")
	}
	return indexes[0].CollateralType
}

func usdxRewardIndexToMultiRewardIndexes(index types.RewardIndexes) types.MultiRewardIndexes {
	var multi types.MultiRewardIndexes
	for _, i := range index {
		// Each collateral type has "ukava" reward index
		multiRewardIndex := types.NewMultiRewardIndex(i.CollateralType, types.RewardIndexes{
			types.NewRewardIndex(types.USDXMintingRewardDenom, i.RewardFactor),
		})

		multi = append(multi, multiRewardIndex)
	}

	return multi
}
