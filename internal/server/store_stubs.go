package server

import (
	"errors"
	"path/filepath"

	"github.com/spacecowboy/feeder-sync/internal/store"
	"github.com/spacecowboy/feeder-sync/internal/store/sqlite"

	"github.com/google/uuid"
)

type InMemoryStore struct {
	calls       map[string]int
	userDevices map[string][]store.UserDevice
	articles    map[uuid.UUID][]store.Article
}

func (s InMemoryStore) RegisterNewUser(deviceName string) (store.UserDevice, error) {
	userId := uuid.New()
	devices := make([]store.UserDevice, 2)

	device := store.UserDevice{
		UserId:         userId,
		DeviceId:       uuid.New(),
		DeviceName:     deviceName,
		LegacySyncCode: userId.String(),
		LegacyDeviceId: 5, //rand.Int63(),
	}

	devices = append(devices, device)
	s.userDevices[userId.String()] = devices

	return device, nil
}

func (s InMemoryStore) AddDeviceToChain(userId uuid.UUID, deviceName string) (store.UserDevice, error) {
	devices := s.userDevices[userId.String()]

	if devices == nil {
		return store.UserDevice{}, errors.New("No such user")
	}

	device := store.UserDevice{
		UserId:     userId,
		DeviceId:   uuid.New(),
		DeviceName: deviceName,
	}

	devices = append(devices, device)
	s.userDevices[userId.String()] = devices

	return device, nil
}

func (s InMemoryStore) AddDeviceToChainWithLegacy(syncCode string, deviceName string) (store.UserDevice, error) {
	devices := s.userDevices[syncCode]

	if devices == nil {
		return store.UserDevice{}, errors.New("No such user")
	}

	device := store.UserDevice{
		UserId:     devices[0].UserId,
		DeviceId:   uuid.New(),
		DeviceName: deviceName,
	}

	devices = append(devices, device)
	s.userDevices[syncCode] = devices

	return device, nil
}

func (s InMemoryStore) GetArticles(userId uuid.UUID, sinceMillis int64) ([]store.Article, error) {
	articles := s.articles[userId]

	if articles == nil {
		// return []store.Article{}, errors.New("No such user")
		return []store.Article{}, nil
	}

	return articles, nil
}

func (s InMemoryStore) AddLegacyArticle(userDbId int64, identifier string) error {
	return errors.New("BOOM")
}

func (s InMemoryStore) GetLegacyDevice(syncCode string, deviceId int64) (store.UserDevice, error) {
	return store.UserDevice{}, errors.New("BOOM")
}

func (s InMemoryStore) Close() error {
	return nil
}

func (s InMemoryStore) EnsureMigration(syncCode string, deviceId int64, deviceName string) (int64, error) {
	s.calls["EnsureMigration"] = 1 + s.calls["EnsureMigration"]
	return 0, nil
}

func (s InMemoryStore) UpdateLastSeenForDevice(device store.UserDevice) (int64, error) {
	return 0, errors.New("BOOM")
}

func (s InMemoryStore) RemoveDeviceWithLegacy(userDbId int64, legacyDeviceId int64) (int64, error) {
	return 0, errors.New("BOOM")
}

func (s InMemoryStore) GetDevices(userId uuid.UUID) ([]store.UserDevice, error) {
	return nil, errors.New("BOOM")
}

func (s InMemoryStore) GetLegacyFeeds(userId uuid.UUID) (store.LegacyFeeds, error) {
	return store.LegacyFeeds{}, errors.New("BOOM")
}

type ExplodingStore struct{}

func (s ExplodingStore) RegisterNewUser(deviceName string) (store.UserDevice, error) {
	return store.UserDevice{}, errors.New("BOOM")
}

func (s ExplodingStore) AddDeviceToChain(userId uuid.UUID, deviceName string) (store.UserDevice, error) {
	return store.UserDevice{}, errors.New("BOOM")
}

func (s ExplodingStore) AddDeviceToChainWithLegacy(syncCode string, deviceName string) (store.UserDevice, error) {
	return store.UserDevice{}, errors.New("BOOM")
}

func (s ExplodingStore) EnsureMigration(syncCode string, deviceId int64, deviceName string) (int64, error) {
	return 0, errors.New("BOOM")
}

func (s ExplodingStore) GetArticles(userId uuid.UUID, sinceMillis int64) ([]store.Article, error) {
	return nil, errors.New("BOOM")
}

func (s ExplodingStore) AddLegacyArticle(userDbId int64, identifier string) error {
	return errors.New("BOOM")
}

func (s ExplodingStore) GetLegacyDevice(syncCode string, deviceId int64) (store.UserDevice, error) {
	return store.UserDevice{}, errors.New("BOOM")
}

func (s ExplodingStore) UpdateLastSeenForDevice(device store.UserDevice) (int64, error) {
	return 0, errors.New("BOOM")
}

func (s ExplodingStore) Close() error {
	return errors.New("BOOM")
}

func (s ExplodingStore) RemoveDeviceWithLegacy(userDbId int64, legacyDeviceId int64) (int64, error) {
	return 0, errors.New("BOOM")
}

func (s ExplodingStore) GetDevices(userId uuid.UUID) ([]store.UserDevice, error) {
	return nil, errors.New("BOOM")
}

func (s ExplodingStore) GetLegacyFeeds(userId uuid.UUID) (store.LegacyFeeds, error) {
	return store.LegacyFeeds{}, errors.New("BOOM")
}

func NewSqliteServer(tempdir string) (*FeederServer, error) {
	store, err := sqlite.New(filepath.Join(tempdir, "sqlite.db"))
	if err != nil {
		return nil, err
	}

	if err := store.RunMigrations("file://../../migrations"); err != nil {
		return nil, err
	}

	return NewServerWithStore(&store)
}
