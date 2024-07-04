package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type DataStore interface {
	Close() error
	RegisterNewUser(ctx context.Context, deviceName string) (UserDevice, error)
	AddDeviceToChain(ctx context.Context, userId uuid.UUID, deviceName string) (UserDevice, error)
	AddDeviceToChainWithLegacy(ctx context.Context, syncCode string, deviceName string) (UserDevice, error)
	GetDevices(ctx context.Context, userId uuid.UUID) ([]UserDevice, error)
	GetLegacyDevice(ctx context.Context, syncCode string, deviceId int64) (UserDevice, error)
	GetLegacyDevicesEtag(ctx context.Context, syncCode string) (string, error)
	RemoveDeviceWithLegacy(ctx context.Context, userDbId int64, legacyDeviceId int64) (int64, error)
	UpdateLastSeenForDevice(ctx context.Context, device UserDevice) (int64, error)
	GetArticles(ctx context.Context, userId uuid.UUID, sinceMillis int64) ([]Article, error)
	AddLegacyArticle(ctx context.Context, userDbId int64, identifier string) error
	GetLegacyFeeds(ctx context.Context, userId uuid.UUID) (LegacyFeeds, error)
	GetLegacyFeedsEtag(ctx context.Context, userId uuid.UUID) (string, error)
	UpdateLegacyFeeds(ctx context.Context, userDbId int64, contentHash int64, content string, etag string) (int64, error)
	// Inserts a new user and device with the given legacy values if not already exists.
	// NOOP if already exists.
	EnsureMigration(ctx context.Context, syncCode string, deviceId int64, deviceName string) (int64, error)
}

type TransferStore interface {
	TransferUsersToStore(ctx context.Context, toStore TransferStore) error
	AcceptUser(ctx context.Context, user *User) error
	TransferDevicesToStore(ctx context.Context, toStore TransferStore) error
	AcceptDevice(ctx context.Context, device *UserDevice) error
	TransferArticlesToStore(ctx context.Context, toStore TransferStore) error
	AcceptArticle(ctx context.Context, article *Article) error
	TransferLegacyFeedsToStore(ctx context.Context, toStore TransferStore) error
	AcceptLegacyFeeds(ctx context.Context, feeds *LegacyFeeds) error
}

type User struct {
	UserDbId       int64
	UserId         uuid.UUID
	LegacySyncCode string
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
	UserDbId   int64
	UserId     uuid.UUID
	ReadTime   int64
	Identifier string
	UpdatedAt  int64
}

type LegacyFeeds struct {
	UserDbId    int64
	UserId      uuid.UUID
	ContentHash int64
	Content     string
	Etag        string
}

var ErrNoFeeds = errors.New("store: no feeds")
var ErrNoSuchDevice = errors.New("store: no such device")
