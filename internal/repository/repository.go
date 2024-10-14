package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/spacecowboy/feeder-sync/build/gen/db"
)

type Repository interface {
	Close(ctx context.Context) error
	RegisterNewUser(deviceName string) (db.User, error)
	AddDeviceToChain(userId uuid.UUID, deviceName string) (UserAndDevice, error)
	AddDeviceToChainWithLegacy(syncCode string, deviceName string) (UserAndDevice, error)
	GetDevices(userId uuid.UUID) ([]UserAndDevice, error)
	GetLegacyDevice(syncCode string, deviceId int64) (UserAndDevice, error)
	GetLegacyDevicesEtag(syncCode string) (string, error)
	RemoveDeviceWithLegacy(userDbId int64, legacyDeviceId int64) (int64, error)
	UpdateLastSeenForDevice(device UserAndDevice) (int64, error)
	GetArticles(userId uuid.UUID, sinceMillis int64) ([]db.Article, error)
	AddLegacyArticle(userDbId int64, identifier string) error
	GetLegacyFeeds(userId uuid.UUID) (db.LegacyFeed, error)
	GetLegacyFeedsEtag(userId uuid.UUID) (string, error)
	UpdateLegacyFeeds(userDbId int64, contentHash int64, content string, etag string) (int64, error)
	// Inserts a new user and device with the given legacy values if not already exists.
	// NOOP if already exists.
	EnsureMigration(syncCode string, deviceId int64, deviceName string) (int64, error)
	// Admin functions
	TransferUsers(repository Repository) error
	AcceptUser(user *db.User) error
	TransferDevices(repository Repository) error
	AcceptDevice(device *UserAndDevice) error
	TransferArticles(repository Repository) error
	AcceptArticle(article *db.Article) error
	TransferLegacyFeeds(repository Repository) error
	AcceptLegacyFeeds(feeds *db.LegacyFeed) error
	// For health check
	PingContext(ctx context.Context) error
}

type UserAndDevice struct {
	User   db.User
	Device db.Device
}

// type User struct {
// 	UserDbId       int64
// 	UserId         uuid.UUID
// 	LegacySyncCode string
// }

// type UserDevice struct {
// 	UserDbId   int64
// 	UserId     uuid.UUID
// 	DeviceId   uuid.UUID
// 	DeviceName string
// 	LastSeen   int64

// 	// Migration fields
// 	LegacySyncCode string
// 	LegacyDeviceId int64
// }

// type Article struct {
// 	UserDbId   int64
// 	UserId     uuid.UUID
// 	ReadTime   int64
// 	Identifier string
// 	UpdatedAt  int64
// }

// type LegacyFeeds struct {
// 	UserDbId    int64
// 	UserId      uuid.UUID
// 	ContentHash int64
// 	Content     string
// 	Etag        string
// }

var ErrNoFeeds = errors.New("repository: no feeds")
var ErrNoSuchDevice = errors.New("repository: no such device")
