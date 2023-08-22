package store

import (
	"github.com/google/uuid"
)

type DataStore interface {
	Close() error
	RegisterNewUser(deviceName string) (UserDevice, error)
	AddDeviceToChain(userId uuid.UUID, deviceName string) (UserDevice, error)
	AddDeviceToChainWithLegacy(syncCode string, deviceName string) (UserDevice, error)
	EnsureMigration(syncCode string, deviceId int64, deviceName string) error
}

type UserDevice struct {
	UserId     uuid.UUID
	DeviceId   uuid.UUID
	DeviceName string

	// Migration fields
	LegacySyncCode string
	LegacyDeviceId int64
}
