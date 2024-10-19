package repository

import (
	"context"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/spacecowboy/feeder-sync/build/gen/db"
)

type PostgresRepository struct {
	pgConn  *pgx.Conn
	queries *db.Queries
}

func NewPostgresRepository(pgConn *pgx.Conn) *PostgresRepository {
	return &PostgresRepository{
		pgConn:  pgConn,
		queries: db.New(pgConn),
	}
}

// Verify interface implementation
var _ Repository = &PostgresRepository{}

func (r *PostgresRepository) Close(ctx context.Context) error {
	return r.pgConn.Close(ctx)
}

func (r *PostgresRepository) RegisterNewUser(ctx context.Context, deviceName string) (UserAndDevice, error) {
	legacySyncCode, err := randomLegacySyncCode()
	if err != nil {
		return UserAndDevice{}, err
	}

	insertUserParams := db.InsertUserParams{
		UserID:         uuid.NewString(),
		LegacySyncCode: legacySyncCode,
	}
	user, err := r.queries.InsertUser(ctx, insertUserParams)
	if err != nil {
		return UserAndDevice{}, err
	}

	device, err := r.queries.InsertDevice(ctx, db.InsertDeviceParams{
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
		return UserAndDevice{}, err
	}

	return UserAndDevice{
		User:   user,
		Device: device,
	}, nil
}

func (r *PostgresRepository) GetUserByUserId(ctx context.Context, userId uuid.UUID) (db.User, error) {
	return r.queries.GetUserByUserId(ctx, userId.String())
}

func (r *PostgresRepository) GetUserBySyncCode(ctx context.Context, syncCode string) (db.User, error) {
	return r.queries.GetUserBySyncCode(ctx, syncCode)
}

func (r *PostgresRepository) AddDeviceToUser(ctx context.Context, user db.User, deviceName string) (db.Device, error) {
	return r.queries.InsertDevice(ctx, db.InsertDeviceParams{
		UserDbID:       user.DbID,
		DeviceID:       uuid.NewString(),
		DeviceName:     deviceName,
		LegacyDeviceID: rand.Int63(),
		LastSeen: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
	})
}

func (r *PostgresRepository) GetDevices(ctx context.Context, user db.User) ([]db.Device, error) {
	return r.queries.GetDevices(ctx, user.DbID)
}

func (r *PostgresRepository) GetDeviceWithLegacyId(ctx context.Context, user db.User, deviceId int64) (db.Device, error) {
	return r.queries.GetLegacyDevice(ctx, db.GetLegacyDeviceParams{
		UserDbID:       user.DbID,
		LegacyDeviceID: deviceId,
	})
}

func (r *PostgresRepository) GetDevicesEtag(ctx context.Context, user db.User) (string, error) {
	etagBytes, err := r.queries.GetLegacyDevicesEtag(ctx, user.DbID)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(etagBytes), nil
}

func (r *PostgresRepository) RemoveDeviceWithLegacyId(ctx context.Context, user db.User, legacyDeviceId int64) (int, error) {
	ids, err := r.queries.DeleteDeviceWithLegacyId(ctx, db.DeleteDeviceWithLegacyIdParams{
		UserDbID:       user.DbID,
		LegacyDeviceID: legacyDeviceId,
	})
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, ErrNoSuchDevice
	}
	return len(ids), err
}

func (r *PostgresRepository) UpdateLastSeenForDevice(ctx context.Context, device db.Device) error {
	return r.queries.UpdateLastSeenForDevice(ctx, db.UpdateLastSeenForDeviceParams{
		LastSeen: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
		DbID: device.DbID,
	})
}

func (r *PostgresRepository) GetArticlesUpdatedSince(ctx context.Context, user db.User, sinceMillis int64) ([]db.Article, error) {
	articles, err := r.queries.GetArticlesUpdatedSince(ctx, db.GetArticlesUpdatedSinceParams{
		UserDbID: user.DbID,
		UpdatedAt: pgtype.Timestamptz{
			Time:  time.UnixMilli(sinceMillis),
			Valid: true,
		},
	})
	if err != nil && err == pgx.ErrNoRows || len(articles) == 0 {
		return articles, ErrNoReadMarks
	}
	return articles, err
}

func (r *PostgresRepository) AddArticle(ctx context.Context, user db.User, identifier string) (db.Article, error) {
	timestamp := pgtype.Timestamptz{
		Time:  time.Now(),
		Valid: true,
	}
	return r.queries.InsertArticle(ctx, db.InsertArticleParams{
		UserDbID:   user.DbID,
		Identifier: identifier,
		UpdatedAt:  timestamp,
		ReadTime:   timestamp,
	})
}

func (r *PostgresRepository) GetLegacyFeeds(ctx context.Context, user db.User) (db.LegacyFeed, error) {
	feeds, err := r.queries.GetLegacyFeeds(ctx, user.DbID)
	if err != nil && err == pgx.ErrNoRows {
		return feeds, ErrNoFeeds
	}
	return feeds, err
}

func (r *PostgresRepository) UpdateLegacyFeeds(ctx context.Context, user db.User, contentHash int64, content string, etag string) (int64, error) {
	return r.queries.UpdateLegacyFeeds(ctx, db.UpdateLegacyFeedsParams{
		UserDbID:    user.DbID,
		ContentHash: contentHash,
		Content:     content,
		Etag:        etag,
	})
}

// func (r *PostgresRepository) EnsureMigration(ctx context.Context, syncCode string, deviceId int64, deviceName string) (int64, error) {

// 	result, err := r.queries.EnsureMigration(ctx, db.EnsureMigrationParams{
// 		LegacySyncCode: syncCode,
// 		LegacyDeviceID: deviceId,
// 		DeviceName:     deviceName,
// 	})
// 	if err != nil {
// 		return 0, err
// 	}
// 	return result.RowsAffected(), nil
// }

func (r *PostgresRepository) TransferUsers(ctx context.Context, repository Repository) error {
	return errors.New("not implemented")
}

func (r *PostgresRepository) AcceptUser(ctx context.Context, user *db.User) error {
	return errors.New("not implemented")
}

func (r *PostgresRepository) TransferDevices(ctx context.Context, repository Repository) error {
	return errors.New("not implemented")
}

func (r *PostgresRepository) AcceptDevice(ctx context.Context, device *UserAndDevice) error {
	return errors.New("not implemented")
}

func (r *PostgresRepository) TransferArticles(ctx context.Context, repository Repository) error {
	return errors.New("not implemented")
}

func (r *PostgresRepository) AcceptArticle(ctx context.Context, article *db.Article) error {
	return errors.New("not implemented")
}

func (r *PostgresRepository) TransferLegacyFeeds(ctx context.Context, repository Repository) error {
	return errors.New("not implemented")
}

func (r *PostgresRepository) AcceptLegacyFeeds(ctx context.Context, feeds *db.LegacyFeed) error {
	return errors.New("not implemented")
}

func (r *PostgresRepository) PingContext(ctx context.Context) error {
	return r.pgConn.Ping(ctx)
}

func randomLegacySyncCode() (string, error) {
	bytes := make([]byte, 30)
	if _, err := crand.Read(bytes); err != nil {
		return "", err
	}
	syncCode := fmt.Sprintf("feed%s", hex.EncodeToString(bytes))

	if got := len(syncCode); got != 64 {
		log.Printf("code was %d long", got)
		return "", fmt.Errorf("Code was %d long not 64", got)
	}
	return syncCode, nil
}
