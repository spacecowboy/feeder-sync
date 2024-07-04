package server

import (
	"context"
	"errors"

	"github.com/spacecowboy/feeder-sync/internal/store"

	"github.com/google/uuid"
)

type ExplodingStore struct{}

func (s ExplodingStore) GetLegacyFeedsEtag(ctx context.Context, userId uuid.UUID) (string, error) {
	return "", errors.New("BOOM")
}

func (s ExplodingStore) GetLegacyDevicesEtag(ctx context.Context, syncCode string) (string, error) {
	return "", errors.New("BOOM")
}

func (s ExplodingStore) RegisterNewUser(ctx context.Context, deviceName string) (store.UserDevice, error) {
	return store.UserDevice{}, errors.New("BOOM")
}

func (s ExplodingStore) AddDeviceToChain(ctx context.Context, userId uuid.UUID, deviceName string) (store.UserDevice, error) {
	return store.UserDevice{}, errors.New("BOOM")
}

func (s ExplodingStore) AddDeviceToChainWithLegacy(ctx context.Context, syncCode string, deviceName string) (store.UserDevice, error) {
	return store.UserDevice{}, errors.New("BOOM")
}

func (s ExplodingStore) EnsureMigration(ctx context.Context, syncCode string, deviceId int64, deviceName string) (int64, error) {
	return 0, errors.New("BOOM")
}

func (s ExplodingStore) GetArticles(ctx context.Context, userId uuid.UUID, sinceMillis int64) ([]store.Article, error) {
	return nil, errors.New("BOOM")
}

func (s ExplodingStore) AddLegacyArticle(ctx context.Context, userDbId int64, identifier string) error {
	return errors.New("BOOM")
}

func (s ExplodingStore) GetLegacyDevice(ctx context.Context, syncCode string, deviceId int64) (store.UserDevice, error) {
	return store.UserDevice{}, errors.New("BOOM")
}

func (s ExplodingStore) UpdateLastSeenForDevice(ctx context.Context, device store.UserDevice) (int64, error) {
	return 0, errors.New("BOOM")
}

func (s ExplodingStore) Close() error {
	return errors.New("BOOM")
}

func (s ExplodingStore) RemoveDeviceWithLegacy(ctx context.Context, userDbId int64, legacyDeviceId int64) (int64, error) {
	return 0, errors.New("BOOM")
}

func (s ExplodingStore) GetDevices(ctx context.Context, userId uuid.UUID) ([]store.UserDevice, error) {
	return nil, errors.New("BOOM")
}

func (s ExplodingStore) GetLegacyFeeds(ctx context.Context, userId uuid.UUID) (store.LegacyFeeds, error) {
	return store.LegacyFeeds{}, errors.New("BOOM")
}

func (s ExplodingStore) UpdateLegacyFeeds(ctx context.Context, userDbId int64, contentHash int64, content string, etag string) (int64, error) {
	return 0, errors.New("BOOM")
}

func (s ExplodingStore) TransferUsersToStore(ctx context.Context, toStore store.DataStore) error {
	return errors.New("BOOM")
}

func (s ExplodingStore) AcceptUser(ctx context.Context, user *store.User) error {
	return errors.New("BOOM")
}

func (s ExplodingStore) TransferDevicesToStore(ctx context.Context, toStore store.DataStore) error {
	return errors.New("BOOM")
}

func (s ExplodingStore) AcceptDevice(ctx context.Context, device *store.UserDevice) error {
	return errors.New("BOOM")
}

func (s ExplodingStore) TransferArticlesToStore(ctx context.Context, toStore store.DataStore) error {
	return errors.New("BOOM")
}

func (s ExplodingStore) AcceptArticle(ctx context.Context, article *store.Article) error {
	return errors.New("BOOM")
}

func (s ExplodingStore) TransferLegacyFeedsToStore(ctx context.Context, toStore store.DataStore) error {
	return errors.New("BOOM")
}

func (s ExplodingStore) AcceptLegacyFeeds(ctx context.Context, feeds *store.LegacyFeeds) error {
	return errors.New("BOOM")
}
