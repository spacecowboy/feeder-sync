package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/spacecowboy/feeder-sync/build/gen/db"
)

type Repository interface {
	Close(ctx context.Context) error
	RegisterNewUser(ctx context.Context, deviceName string) (UserAndDevice, error)
	AddDeviceToUser(ctx context.Context, user db.User, deviceName string) (db.Device, error)
	GetDevices(ctx context.Context, user db.User) ([]db.Device, error)
	// GetLegacyDevice(ctx context.Context, syncCode string, deviceId int64) (UserAndDevice, error)
	GetDevicesEtag(ctx context.Context, user db.User) (string, error)
	// RemoveDeviceWithLegacy(ctx context.Context, userDbId int64, legacyDeviceId int64) (int64, error)
	UpdateLastSeenForDevice(ctx context.Context, device db.Device) error
	GetArticlesUpdatedSince(ctx context.Context, user db.User, sinceMillis int64) ([]db.Article, error)
	AddArticle(ctx context.Context, user db.User, identifier string) (db.Article, error)
	// AddLegacyArticle(ctx context.Context, userDbId int64, identifier string) error
	GetLegacyFeeds(ctx context.Context, user db.User) (db.LegacyFeed, error)
	UpdateLegacyFeeds(ctx context.Context, user db.User, contentHash int64, content string, etag string) (int64, error)
	GetUserByUserId(ctx context.Context, userId uuid.UUID) (db.User, error)
	GetUserBySyncCode(ctx context.Context, syncCode string) (db.User, error)
	GetDeviceWithLegacyId(ctx context.Context, user db.User, legacyDeviceId int64) (db.Device, error)
	RemoveDeviceWithLegacyId(ctx context.Context, user db.User, legacyDeviceId int64) (int, error)

	// Inserts a new user and device with the given legacy values if not already exists.
	// NOOP if already exists.
	// EnsureMigration(ctx context.Context, syncCode string, deviceId int64, deviceName string) (int64, error)

	// Admin functions
	TransferUsers(ctx context.Context, repository Repository) error
	AcceptUser(ctx context.Context, user *db.User) error
	TransferDevices(ctx context.Context, repository Repository) error
	AcceptDevice(ctx context.Context, device *UserAndDevice) error
	TransferArticles(ctx context.Context, repository Repository) error
	AcceptArticle(ctx context.Context, article *db.Article) error
	TransferLegacyFeeds(ctx context.Context, repository Repository) error
	AcceptLegacyFeeds(ctx context.Context, feeds *db.LegacyFeed) error
	// For health check
	PingContext(ctx context.Context) error
}

type UserAndDevice struct {
	User   db.User
	Device db.Device
}

var ErrNoReadMarks = errors.New("repository: no read marks")
var ErrNoFeeds = errors.New("repository: no feeds")
var ErrNoSuchDevice = errors.New("repository: no such device")
