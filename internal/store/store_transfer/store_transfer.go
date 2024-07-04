package store_transfer

import (
	"context"

	"github.com/spacecowboy/feeder-sync/internal/store"
)

func MoveBetweenStores(ctx context.Context, from_store store.TransferStore, toStore store.TransferStore) error {
	if err := from_store.TransferUsersToStore(ctx, toStore); err != nil {
		return err
	}

	if err := from_store.TransferDevicesToStore(ctx, toStore); err != nil {
		return err
	}

	if err := from_store.TransferArticlesToStore(ctx, toStore); err != nil {
		return err
	}

	if err := from_store.TransferLegacyFeedsToStore(ctx, toStore); err != nil {
		return err
	}

	return nil
}
