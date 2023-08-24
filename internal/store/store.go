package store

import (
	"github.com/google/uuid"
)

type DataStore interface {
	Close() error
	RegisterNewUser(deviceName string) (UserDevice, error)
	AddDeviceToChain(userId uuid.UUID, deviceName string) (UserDevice, error)
	AddDeviceToChainWithLegacy(syncCode string, deviceName string) (UserDevice, error)
	// Inserts a new user and device with the given legacy values if not already exists.
	// NOOP if already exists.
	EnsureMigration(syncCode string, deviceId int64, deviceName string) (int64, error)
}

type UserDevice struct {
	UserId     uuid.UUID
	DeviceId   uuid.UUID
	DeviceName string
	// LastSeen

	// Migration fields
	LegacySyncCode string
	LegacyDeviceId int64
}
