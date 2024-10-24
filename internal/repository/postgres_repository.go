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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spacecowboy/feeder-sync/build/gen/db"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool: pool,
	}
}

// Verify interface implementation
var _ Repository = &PostgresRepository{}

// Acquire a connection from the pool and return a Queries object
// Caller must call the release function to release the connection back to the pool
func (r *PostgresRepository) queries(ctx context.Context) (*db.Queries, func(), error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		log.Printf("failed to acquire connection: %v", err)
		return nil, nil, err
	}
	return db.New(conn), conn.Release, nil
}

func (r *PostgresRepository) Close(ctx context.Context) error {
	r.pool.Close()
	return nil
}

func (r *PostgresRepository) RegisterNewUser(ctx context.Context, deviceName string) (UserAndDevice, error) {
	legacySyncCode, err := randomLegacySyncCode()
	if err != nil {
		return UserAndDevice{}, err
	}

	queries, release, err := r.queries(ctx)
	if err != nil {
		return UserAndDevice{}, err
	}
	defer release()

	insertUserParams := db.InsertUserParams{
		UserID:         uuid.NewString(),
		LegacySyncCode: legacySyncCode,
	}
	user, err := queries.InsertUser(ctx, insertUserParams)
	if err != nil {
		return UserAndDevice{}, err
	}

	device, err := queries.InsertDevice(ctx, db.InsertDeviceParams{
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
	queries, release, err := r.queries(ctx)
	if err != nil {
		return db.User{}, err
	}
	defer release()

	return queries.GetUserByUserId(ctx, userId.String())
}

func (r *PostgresRepository) GetUserBySyncCode(ctx context.Context, syncCode string) (db.User, error) {
	queries, release, err := r.queries(ctx)
	if err != nil {
		return db.User{}, err
	}
	defer release()

	return queries.GetUserBySyncCode(ctx, syncCode)
}

func (r *PostgresRepository) AddDeviceToUser(ctx context.Context, user db.User, deviceName string) (db.Device, error) {
	queries, release, err := r.queries(ctx)
	if err != nil {
		return db.Device{}, err
	}
	defer release()

	return queries.InsertDevice(ctx, db.InsertDeviceParams{
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
	queries, release, err := r.queries(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	return queries.GetDevices(ctx, user.DbID)
}

func (r *PostgresRepository) GetDeviceWithLegacyId(ctx context.Context, user db.User, deviceId int64) (db.Device, error) {
	queries, release, err := r.queries(ctx)
	if err != nil {
		return db.Device{}, err
	}
	defer release()

	return queries.GetLegacyDevice(ctx, db.GetLegacyDeviceParams{
		UserDbID:       user.DbID,
		LegacyDeviceID: deviceId,
	})
}

func (r *PostgresRepository) GetDevicesEtag(ctx context.Context, user db.User) (string, error) {
	queries, release, err := r.queries(ctx)
	if err != nil {
		return "", err
	}
	defer release()

	etagBytes, err := queries.GetLegacyDevicesEtag(ctx, user.DbID)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(etagBytes), nil
}

func (r *PostgresRepository) RemoveDeviceWithLegacyId(ctx context.Context, user db.User, legacyDeviceId int64) (int, error) {
	queries, release, err := r.queries(ctx)
	if err != nil {
		return 0, err
	}
	defer release()

	ids, err := queries.DeleteDeviceWithLegacyId(ctx, db.DeleteDeviceWithLegacyIdParams{
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
	queries, release, err := r.queries(ctx)
	if err != nil {
		return err
	}
	defer release()

	return queries.UpdateLastSeenForDevice(ctx, db.UpdateLastSeenForDeviceParams{
		LastSeen: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
		DbID: device.DbID,
	})
}

func (r *PostgresRepository) GetArticlesUpdatedSince(ctx context.Context, user db.User, sinceMillis int64) ([]db.Article, error) {
	queries, release, err := r.queries(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	articles, err := queries.GetArticlesUpdatedSince(ctx, db.GetArticlesUpdatedSinceParams{
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
	queries, release, err := r.queries(ctx)
	if err != nil {
		return db.Article{}, err
	}
	defer release()

	timestamp := pgtype.Timestamptz{
		Time:  time.Now(),
		Valid: true,
	}
	return queries.InsertArticle(ctx, db.InsertArticleParams{
		UserDbID:   user.DbID,
		Identifier: identifier,
		UpdatedAt:  timestamp,
		ReadTime:   timestamp,
	})
}

func (r *PostgresRepository) GetLegacyFeeds(ctx context.Context, user db.User) (db.LegacyFeed, error) {
	queries, release, err := r.queries(ctx)
	if err != nil {
		return db.LegacyFeed{}, err
	}
	defer release()

	feeds, err := queries.GetLegacyFeeds(ctx, user.DbID)
	if err != nil && err == pgx.ErrNoRows {
		return feeds, ErrNoFeeds
	}
	return feeds, err
}

func (r *PostgresRepository) UpdateLegacyFeeds(ctx context.Context, user db.User, contentHash int64, content string, etag string) (int64, error) {
	queries, release, err := r.queries(ctx)
	if err != nil {
		return 0, err
	}
	defer release()

	return queries.UpdateLegacyFeeds(ctx, db.UpdateLegacyFeedsParams{
		UserDbID:    user.DbID,
		ContentHash: contentHash,
		Content:     content,
		Etag:        etag,
	})
}

// func (r *PostgresRepository) EnsureMigration(ctx context.Context, syncCode string, deviceId int64, deviceName string) (int64, error) {

// 	result, err := queries.EnsureMigration(ctx, db.EnsureMigrationParams{
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
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	return conn.Ping(ctx)
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
