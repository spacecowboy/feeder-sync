package store_transfer

import (
	"errors"

	"github.com/spacecowboy/feeder-sync/internal/repository"
)

func MoveBetweenRepositories(from repository.Repository, to repository.Repository) error {
	return errors.New("not implemented")
	// if err := from.TransferUsersToStore(to); err != nil {
	// 	return err
	// }

	// if err := from.TransferDevicesToStore(to); err != nil {
	// 	return err
	// }

	// if err := from_store.TransferArticlesToStore(toStore); err != nil {
	// 	return err
	// }

	// if err := from.TransferLegacyFeedsToStore(to); err != nil {
	// 	return err
	// }

	// return nil
}
