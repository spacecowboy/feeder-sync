package server

import (
	"errors"

	"github.com/spacecowboy/feeder-sync/internal/store"

	"github.com/google/uuid"
)

type InMemoryStore struct {
	calls       map[string]int
	userDevices map[string][]store.UserDevice
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

func (s InMemoryStore) Close() error {
	return nil
}

func (s InMemoryStore) EnsureMigration(syncCode string, deviceId int64, deviceName string) (int64, error) {
	s.calls["EnsureMigration"] = 1 + s.calls["EnsureMigration"]
	return 0, nil
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

func (s ExplodingStore) Close() error {
	return errors.New("BOOM")
}
