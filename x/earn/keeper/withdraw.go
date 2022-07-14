package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/kava-labs/kava/x/earn/types"
)

func (k *Keeper) Withdraw(ctx sdk.Context, from sdk.AccAddress, amount sdk.Coin) error {
	// Get AllowedVault, if not found (not a valid vault), return error
	allowedVault, found := k.GetAllowedVault(ctx, amount.Denom)
	if !found {
		return types.ErrInvalidVaultDenom
	}

	if amount.IsZero() {
		return types.ErrInsufficientAmount
	}

	// Check if VaultRecord exists, return error if not exist as it's empty
	vaultRecord, found := k.GetVaultRecord(ctx, amount.Denom)
	if !found {
		return types.ErrVaultRecordNotFound
	}

	// Get VaultShareRecord for account, create if not exist
	vaultShareRecord, found := k.GetVaultShareRecord(ctx, amount.Denom, from)
	if !found {
		return types.ErrVaultShareRecordNotFound
	}

	// Check if VaultShareRecord has enough supplied to withdraw
	if vaultShareRecord.AmountSupplied.Amount.LT(amount.Amount) {
		return sdkerrors.Wrapf(
			types.ErrInvalidShares,
			"withdraw of %s shares greater than %s shares supplied",
			amount,
			vaultShareRecord.AmountSupplied,
		)
	}

	// Decrement VaultRecord and VaultShareRecord supplies
	vaultRecord.TotalSupply = vaultRecord.TotalSupply.Sub(amount)
	vaultShareRecord.AmountSupplied = vaultShareRecord.AmountSupplied.Sub(amount)

	// Update VaultRecord and VaultShareRecord
	// TODO: Delete records if supplies are zero
	k.SetVaultRecord(ctx, vaultRecord)
	k.SetVaultShareRecord(ctx, vaultShareRecord)

	// Get the strategy for the vault
	strategy, err := k.GetStrategy(allowedVault.VaultStrategy)
	if err != nil {
		return err
	}

	// Deposit to the strategy
	if err := strategy.Withdraw(amount); err != nil {
		return err
	}

	return nil
}
