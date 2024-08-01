package store_transfer

import (
	"github.com/spacecowboy/feeder-sync/internal/store"
)

func MoveBetweenStores(from_store store.DataStore, toStore store.DataStore) error {
	if err := from_store.TransferUsersToStore(toStore); err != nil {
		return err
	}

	if err := from_store.TransferDevicesToStore(toStore); err != nil {
		return err
	}

	// if err := from_store.TransferArticlesToStore(toStore); err != nil {
	// 	return err
	// }

	if err := from_store.TransferLegacyFeedsToStore(toStore); err != nil {
		return err
	}

	return nil
}
