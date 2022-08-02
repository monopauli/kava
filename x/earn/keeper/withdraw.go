package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/kava-labs/kava/x/earn/types"
)

// Withdraw removes the amount of supplied tokens from a vault and transfers it
// back to the account.
func (k *Keeper) Withdraw(ctx sdk.Context, from sdk.AccAddress, wantAmount sdk.Coin) error {
	// Get AllowedVault, if not found (not a valid vault), return error
	allowedVault, found := k.GetAllowedVault(ctx, wantAmount.Denom)
	if !found {
		return types.ErrInvalidVaultDenom
	}

	if wantAmount.IsZero() {
		return types.ErrInsufficientAmount
	}

	// Check if VaultRecord exists
	vaultRecord, found := k.GetVaultRecord(ctx, wantAmount.Denom)
	if !found {
		return types.ErrVaultRecordNotFound
	}

	// Get account share record for the vault
	vaultShareRecord, found := k.GetVaultShareRecord(ctx, from)
	if !found {
		return types.ErrVaultShareRecordNotFound
	}

	withdrawShares, err := k.ConvertToShares(ctx, wantAmount)
	if err != nil {
		return fmt.Errorf("failed to convert assets to shares: %w", err)
	}

	accCurrentShares := vaultShareRecord.Shares.AmountOf(wantAmount.Denom)
	// Check if account is not withdrawing more shares than they have
	if accCurrentShares.LT(withdrawShares.Amount) {
		return sdkerrors.Wrapf(
			types.ErrInsufficientValue,
			"account has less %s vault shares than withdraw shares, %s < %s",
			wantAmount.Denom,
			accCurrentShares,
			withdrawShares.Amount,
		)
	}

	// Convert shares to amount to get truncated true share value
	withdrawAmount, err := k.ConvertToAssets(ctx, withdrawShares)
	if err != nil {
		return fmt.Errorf("failed to convert shares to assets: %w", err)
	}

	accountValue, err := k.GetVaultAccountValue(ctx, wantAmount.Denom, from)
	if err != nil {
		return fmt.Errorf("failed to get account value: %w", err)
	}

	// Check if withdrawAmount > account value
	if withdrawAmount.Amount.GT(accountValue.Amount) {
		return sdkerrors.Wrapf(
			types.ErrInsufficientValue,
			"account has less %s vault value than withdraw amount, %s < %s",
			withdrawAmount.Denom,
			accountValue.Amount,
			withdrawAmount.Amount,
		)
	}

	// Get the strategy for the vault
	strategy, err := k.GetStrategy(allowedVault.VaultStrategy)
	if err != nil {
		return err
	}

	// Not necessary to check if amount denom is allowed for the strategy, as
	// there would be no vault record if it weren't allowed.

	// Withdraw the withdrawAmount from the strategy
	if err := strategy.Withdraw(ctx, withdrawAmount); err != nil {
		return fmt.Errorf("failed to withdraw from strategy: %w", err)
	}

	// Send coins back to account, must withdraw from strategy first or the
	// module account may not have any funds to send.
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		types.ModuleName,
		from,
		sdk.NewCoins(withdrawAmount),
	); err != nil {
		return err
	}

	// Check if new account balance of shares results in account share value
	// of < 1 of a sdk.Coin. This share value is not able to be withdrawn and
	// should just be removed.
	isDust, err := k.ShareIsDust(
		ctx,
		vaultShareRecord.Shares.GetShare(withdrawAmount.Denom).Sub(withdrawShares),
	)
	if err != nil {
		return err
	}

	if isDust {
		// Modify withdrawShares to subtract entire share balance for denom
		withdrawShares = vaultShareRecord.Shares.GetShare(withdrawAmount.Denom)
	}

	// Decrement VaultRecord and VaultShareRecord supplies
	vaultShareRecord.Shares = vaultShareRecord.Shares.Sub(withdrawShares)
	vaultRecord.TotalShares = vaultRecord.TotalShares.Sub(withdrawShares)

	// Update VaultRecord and VaultShareRecord, deletes if zero supply
	k.UpdateVaultRecord(ctx, vaultRecord)
	k.UpdateVaultShareRecord(ctx, vaultShareRecord)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeVaultWithdraw,
			sdk.NewAttribute(types.AttributeKeyVaultDenom, withdrawAmount.Denom),
			sdk.NewAttribute(types.AttributeKeyOwner, from.String()),
			sdk.NewAttribute(types.AttributeKeyShares, withdrawShares.Amount.String()),
			sdk.NewAttribute(sdk.AttributeKeyAmount, withdrawAmount.Amount.String()),
		),
	)

	return nil
}
