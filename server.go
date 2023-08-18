package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

type FeederServer struct {
	store DataStore
}

// server.go
func (s *FeederServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	router := http.NewServeMux()
	router.Handle("/api/v1/create", http.HandlerFunc(s.handleCreate))
	router.Handle("/api/v1/join", http.HandlerFunc(s.handleJoin))

	router.ServeHTTP(w, r)
}

func (s *FeederServer) handleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		http.Error(w, "No body", http.StatusBadRequest)
		return
	}

	var createChainRequest CreateChainRequest

	if err := json.NewDecoder(r.Body).Decode(&createChainRequest); err != nil {
		http.Error(w, "Bad body", http.StatusBadRequest)
		return
	}

	userDevice, err := s.store.RegisterNewUser(createChainRequest.DeviceName)
	if err != nil {
		http.Error(w, "Badness", http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(userDevice); err != nil {
		http.Error(w, "Could not encode UserDevice", http.StatusInternalServerError)
		return
	}
}

func (s *FeederServer) handleJoin(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		http.Error(w, "No body", http.StatusBadRequest)
		return
	}

	var joinChainRequest JoinChainRequest

	if err := json.NewDecoder(r.Body).Decode(&joinChainRequest); err != nil {
		http.Error(w, "Bad body", http.StatusBadRequest)
		return
	}

	userDevice, err := s.store.AddDeviceToChain(joinChainRequest.UserId, joinChainRequest.DeviceName)
	if err != nil {
		switch err.Error() {
		case "No such user":
			http.Error(w, "No such chain", http.StatusNotFound)
		default:
			http.Error(w, "Badness", http.StatusInternalServerError)
		}
		return
	}

	// http.Error(w, "No such chain", http.StatusNotFound)
	if err := json.NewEncoder(w).Encode(userDevice); err != nil {
		http.Error(w, "Could not encode UserDevice", http.StatusInternalServerError)
		return
	}
}

type DataStore interface {
	RegisterNewUser(deviceName string) (UserDevice, error)
	AddDeviceToChain(userId uuid.UUID, deviceName string) (UserDevice, error)
}

type InMemoryStore struct {
	userDevices map[uuid.UUID][]UserDevice
}

func (s InMemoryStore) RegisterNewUser(deviceName string) (UserDevice, error) {
	userId := uuid.New()
	var devices []UserDevice
	devices = make([]UserDevice, 2)

	device := UserDevice{
		UserId:     userId,
		DeviceId:   uuid.New(),
		DeviceName: deviceName,
	}

	devices = append(devices, device)
	s.userDevices[userId] = devices

	return device, nil
}

func (s InMemoryStore) AddDeviceToChain(userId uuid.UUID, deviceName string) (UserDevice, error) {
	var devices []UserDevice
	devices = s.userDevices[userId]

	if devices == nil {
		return UserDevice{}, errors.New("No such user")
	}

	device := UserDevice{
		UserId:     userId,
		DeviceId:   uuid.New(),
		DeviceName: deviceName,
	}

	devices = append(devices, device)
	s.userDevices[userId] = devices

	return device, nil
}

type UserDevice struct {
	UserId     uuid.UUID
	DeviceId   uuid.UUID
	DeviceName string
}

type CreateChainRequest struct {
	DeviceName string
}

type JoinChainRequest struct {
	UserId     uuid.UUID
	DeviceName string
}
