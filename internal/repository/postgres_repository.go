package repository

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/spacecowboy/feeder-sync/build/gen/db"
)

type PostgresRepository struct {
	pgConn *pgx.Conn
}

// Verify interface implementation
var _ Repository = &PostgresRepository{}

func (r *PostgresRepository) Close(ctx context.Context) error {
	return r.pgConn.Close(ctx)
}

func (r *PostgresRepository) RegisterNewUser(deviceName string) (db.User, error) {
	ctx := context.Background()
	legacySyncCode, err := randomLegacySyncCode()
	if err != nil {
		return db.User{}, err
	}

	insertUserParams := db.InsertUserParams{
		UserID:         uuid.NewString(),
		LegacySyncCode: legacySyncCode,
	}
	user, err := db.New(r.pgConn).InsertUser(ctx, insertUserParams)
	if err != nil {
		return db.User{}, err
	}

	_, err = db.New(r.pgConn).InsertDevice(ctx, db.InsertDeviceParams{
		UserDbID:       user.DbID,
		DeviceID:       uuid.NewString(),
		DeviceName:     deviceName,
		LegacyDeviceID: rand.Int63(),
		LastSeen: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
	})
	if err != nil {
		return db.User{}, err
	}

	return user, nil
}

func (r *PostgresRepository) AddDeviceToChain(userId uuid.UUID, deviceName string) (store.UserAndDevice, error) {
	ctx := context.Background()
	user, err := db.New(r.pgConn).GetUserByUserId(ctx, userId.String())
	if err != nil {
		return store.UserAndDevice{}, err
	}

	device, err := db.New(r.pgConn).InsertDevice(ctx, db.InsertDeviceParams{
		UserDbID:       user.DbID,
		DeviceID:       uuid.NewString(),
		DeviceName:     deviceName,
		LegacyDeviceID: rand.Int63(),
		LastSeen: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
	})
	if err != nil {
		return store.UserAndDevice{}, err
	}

	return store.UserAndDevice{User: user, Device: device}, nil
}

func (r *PostgresRepository) AddDeviceToChainWithLegacy(syncCode string, deviceName string) (store.UserAndDevice, error) {
	ctx := context.Background()
	user, err := db.New(r.pgConn).GetUserBySyncCode(ctx, syncCode)
	if err != nil {
		return store.UserAndDevice{}, err
	}

	device, err := db.New(r.pgConn).InsertDevice(ctx, db.InsertDeviceParams{
		UserDbID:       user.DbID,
		DeviceID:       uuid.NewString(),
		DeviceName:     deviceName,
		LegacyDeviceID: rand.Int63(),
		LastSeen: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
	})
	if err != nil {
		return store.UserAndDevice{}, err
	}

	return store.UserAndDevice{User: user, Device: device}, nil
}

func (r *PostgresRepository) GetDevices(userId uuid.UUID) ([]store.UserAndDevice, error) {
	ctx := context.Background()
	devices, err := db.New(r.pgConn).GetDevices(ctx, userId.String())
	if err != nil {
		return nil, err
	}

	var result []store.UserAndDevice
	for _, device := range devices {
		result = append(result, store.UserAndDevice{
			User:   db.User{DbID: device.UserDbID},
			Device: device,
		})
	}

	return result, nil
}

func (r *PostgresRepository) GetLegacyDevice(syncCode string, deviceId int64) (store.UserAndDevice, error) {
	ctx := context.Background()
	legacyDeviceRow, err := db.New(r.pgConn).GetLegacyDevice(ctx, db.GetLegacyDeviceParams{
		LegacySyncCode: syncCode,
		LegacyDeviceID: deviceId,
	})
	if err != nil {
		return store.UserAndDevice{}, err
	}

	return store.UserAndDevice{User: legacyDeviceRow.User, Device: legacyDeviceRow.Device}, nil
}

func (r *PostgresRepository) GetLegacyDevicesEtag(syncCode string) (string, error) {
	ctx := context.Background()
	etagBytes, err := db.New(r.pgConn).GetLegacyDevicesEtag(ctx, syncCode)
	if err != nil {
		return "", err
	}
	return string(etagBytes), nil
}

func (r *PostgresRepository) RemoveDeviceWithLegacy(userDbId int64, legacyDeviceId int64) (int64, error) {
	ctx := context.Background()
	result, err := db.New(r.pgConn).DeleteDeviceWithLegacyId(ctx, db.DeleteDeviceWithLegacyIdParams{
		UserDbID:       userDbId,
		LegacyDeviceID: legacyDeviceId,
	})
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func (r *PostgresRepository) UpdateLastSeenForDevice(device store.UserAndDevice) (int64, error) {
	ctx := context.Background()
	result, err := db.New(r.pgConn).UpdateLastSeenForDevice(ctx, db.UpdateLastSeenForDeviceParams{
		LastSeen: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
		DbID: device.Device.DbID,
	})
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func (r *PostgresRepository) GetArticles(userId uuid.UUID, sinceMillis int64) ([]db.Article, error) {
	ctx := context.Background()
	articles, err := db.New(r.pgConn).GetArticles(ctx, db.GetArticlesParams{
		UserID:      userId.String(),
		SinceMillis: sinceMillis,
	})
	if err != nil {
		return nil, err
	}
	return articles, nil
}

func (r *PostgresRepository) AddLegacyArticle(userDbId int64, identifier string) error {
	ctx := context.Background()
	err := db.New(r.pgConn).InsertLegacyArticle(ctx, db.InsertLegacyArticleParams{
		UserDbID:   userDbId,
		Identifier: identifier,
	})
	return err
}

func (r *PostgresRepository) GetLegacyFeeds(userId uuid.UUID) (db.LegacyFeed, error) {
	ctx := context.Background()
	feeds, err := db.New(r.pgConn).GetLegacyFeeds(ctx, userId.String())
	if err != nil {
		return db.LegacyFeed{}, err
	}
	return feeds, nil
}

func (r *PostgresRepository) GetLegacyFeedsEtag(userId uuid.UUID) (string, error) {
	ctx := context.Background()
	etag, err := db.New(r.pgConn).GetLegacyFeedsEtag(ctx, userId.String())
	if err != nil {
		return "", err
	}
	return etag, nil
}

func (r *PostgresRepository) UpdateLegacyFeeds(userDbId int64, contentHash int64, content string, etag string) (int64, error) {
	ctx := context.Background()
	result, err := db.New(r.pgConn).UpdateLegacyFeeds(ctx, db.UpdateLegacyFeedsParams{
		UserDbID:    userDbId,
		ContentHash: contentHash,
		Content:     content,
		Etag:        etag,
	})
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func (r *PostgresRepository) EnsureMigration(syncCode string, deviceId int64, deviceName string) (int64, error) {
	ctx := context.Background()
	result, err := db.New(r.pgConn).EnsureMigration(ctx, db.EnsureMigrationParams{
		LegacySyncCode: syncCode,
		LegacyDeviceID: deviceId,
		DeviceName:     deviceName,
	})
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func (r *PostgresRepository) TransferUsers(repository store.Repository) error {
	return errors.New("not implemented")
}

func (r *PostgresRepository) AcceptUser(user *db.User) error {
	return errors.New("not implemented")
}

func (r *PostgresRepository) TransferDevices(repository store.Repository) error {
	return errors.New("not implemented")
}

func (r *PostgresRepository) AcceptDevice(device *store.UserAndDevice) error {
	return errors.New("not implemented")
}

func (r *PostgresRepository) TransferArticles(repository store.Repository) error {
	return errors.New("not implemented")
}

func (r *PostgresRepository) AcceptArticle(article *db.Article) error {
	return errors.New("not implemented")
}

func (r *PostgresRepository) TransferLegacyFeeds(repository store.Repository) error {
	return errors.New("not implemented")
}

func (r *PostgresRepository) AcceptLegacyFeeds(feeds *db.LegacyFeed) error {
	return errors.New("not implemented")
}

func (r *PostgresRepository) PingContext(ctx context.Context) error {
	return r.pgConn.Ping(ctx)
}

func randomLegacySyncCode() (string, error) {
	// Implement the logic to generate a random legacy sync code
	return "randomSyncCode", nil
}
