package store

import (
	"errors"

	"github.com/google/uuid"
)

type DataStore interface {
	Close() error
	RegisterNewUser(deviceName string) (UserDevice, error)
	AddDeviceToChain(userId uuid.UUID, deviceName string) (UserDevice, error)
	AddDeviceToChainWithLegacy(syncCode string, deviceName string) (UserDevice, error)
	GetDevices(userId uuid.UUID) ([]UserDevice, error)
	GetLegacyDevice(syncCode string, deviceId int64) (UserDevice, error)
	RemoveDeviceWithLegacy(userDbId int64, legacyDeviceId int64) (int64, error)
	UpdateLastSeenForDevice(device UserDevice) (int64, error)
	GetArticles(userId uuid.UUID, sinceMillis int64) ([]Article, error)
	AddLegacyArticle(userDbId int64, identifier string) error
	GetLegacyFeeds(userId uuid.UUID) (LegacyFeeds, error)
	UpdateLegacyFeeds(userDbId int64, contentHash int64, content string, etag string) (int64, error)
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

type LegacyFeeds struct {
	UserId      uuid.UUID
	ContentHash int64
	Content     string
	Etag        string
}

var ErrNoFeeds = errors.New("store: no feeds")
