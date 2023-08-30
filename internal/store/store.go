package store

import (
	"github.com/google/uuid"
)

type DataStore interface {
	Close() error
	RegisterNewUser(deviceName string) (UserDevice, error)
	AddDeviceToChain(userId uuid.UUID, deviceName string) (UserDevice, error)
	AddDeviceToChainWithLegacy(syncCode string, deviceName string) (UserDevice, error)
	GetLegacyDevice(syncCode string, deviceId int64) (UserDevice, error)
	UpdateLastSeenForDevice(device UserDevice) (int64, error)
	GetArticles(userId uuid.UUID, sinceMillis int64) ([]Article, error)
	AddLegacyArticle(userDbId int64, identifier string) error
	// Inserts a new user and device with the given legacy values if not already exists.
	// NOOP if already exists.
	EnsureMigration(syncCode string, deviceId int64, deviceName string) (int64, error)
}

type UserDevice struct {
	UserDbId   int64
	UserId     uuid.UUID
	DeviceId   uuid.UUID
	DeviceName string
	LastSeen   int64

	// Migration fields
	LegacySyncCode string
	LegacyDeviceId int64
}

type Article struct {
	UserId     uuid.UUID
	ReadTime   int64
	Identifier string
	UpdatedAt  int64
}
