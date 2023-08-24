package server

import (
	"encoding/json"
	"net/http"

	"github.com/spacecowboy/feeder-sync/internal/store"
	"github.com/spacecowboy/feeder-sync/internal/store/sqlite"

	"github.com/google/uuid"
)

type FeederServer struct {
	store store.DataStore
}

func NewServer() (FeederServer, error) {
	store, err := sqlite.New("./sqlite.db")
	if err != nil {
		return FeederServer{}, err
	}

	if err := store.RunMigrations("file://./migrations"); err != nil {
		return FeederServer{}, err
	}

	return FeederServer{
		store: store,
	}, nil
}

func (s *FeederServer) Close() error {
	return s.store.Close()
}

func (s *FeederServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	router := http.NewServeMux()
	router.Handle("/api/v2/migrate", http.HandlerFunc(s.handleMigrateV2))
	router.Handle("/api/v1/create", http.HandlerFunc(s.handleCreateV1))
	router.Handle("/api/v2/create", http.HandlerFunc(s.handleCreateV2))
	router.Handle("/api/v1/join", http.HandlerFunc(s.handleJoinV1))
	router.Handle("/api/v2/join", http.HandlerFunc(s.handleJoinV2))

	router.ServeHTTP(w, r)
}

func (s *FeederServer) handleMigrateV2(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		http.Error(w, "No body", http.StatusBadRequest)
		return
	}

	var migrateRequest MigrateRequestV2

	if err := json.NewDecoder(r.Body).Decode(&migrateRequest); err != nil {
		http.Error(w, "Bad body", http.StatusBadRequest)
		return
	}

	_, err := s.store.EnsureMigration(migrateRequest.SyncCode, migrateRequest.DeviceId, migrateRequest.DeviceName)
	if err != nil {
		http.Error(w, "Badness", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *FeederServer) handleCreateV1(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		http.Error(w, "No body", http.StatusBadRequest)
		return
	}

	var createChainRequest CreateChainRequestV1

	if err := json.NewDecoder(r.Body).Decode(&createChainRequest); err != nil {
		http.Error(w, "Bad body", http.StatusBadRequest)
		return
	}

	userDevice, err := s.store.RegisterNewUser(createChainRequest.DeviceName)
	if err != nil {
		http.Error(w, "Badness", http.StatusInternalServerError)
		return
	}

	response := JoinChainResponseV1{
		SyncCode: userDevice.LegacySyncCode,
		DeviceId: userDevice.LegacyDeviceId,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Could not encode JoinChainResponseV1", http.StatusInternalServerError)
		return
	}
}

func (s *FeederServer) handleCreateV2(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		http.Error(w, "No body", http.StatusBadRequest)
		return
	}

	var createChainRequest CreateChainRequestV2

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

func (s *FeederServer) handleJoinV1(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		http.Error(w, "No body", http.StatusBadRequest)
		return
	}

	syncCode := r.Header.Get("X-FEEDER-ID")
	if syncCode == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
	}

	var joinChainRequest JoinChainRequestV1

	if err := json.NewDecoder(r.Body).Decode(&joinChainRequest); err != nil {
		http.Error(w, "Bad body", http.StatusBadRequest)
		return
	}

	userDevice, err := s.store.AddDeviceToChainWithLegacy(syncCode, joinChainRequest.DeviceName)
	if err != nil {
		switch err.Error() {
		case "No such user":
			http.Error(w, "No such chain", http.StatusNotFound)
		default:
			http.Error(w, "Badness", http.StatusInternalServerError)
		}
		return
	}

	response := JoinChainResponseV1{
		SyncCode: userDevice.LegacySyncCode,
		DeviceId: userDevice.LegacyDeviceId,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Could not encode JoinChainResponseV1", http.StatusInternalServerError)
		return
	}
}

func (s *FeederServer) handleJoinV2(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		http.Error(w, "No body", http.StatusBadRequest)
		return
	}

	var joinChainRequest JoinChainRequestV2

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

	if err := json.NewEncoder(w).Encode(userDevice); err != nil {
		http.Error(w, "Could not encode UserDevice", http.StatusInternalServerError)
		return
	}
}

type CreateChainRequestV1 struct {
	DeviceName string `json:"deviceName"`
}

type JoinChainRequestV1 struct {
	DeviceName string `json:"deviceName"`
}

type JoinChainResponseV1 struct {
	SyncCode string `json:"syncCode"`
	DeviceId int64  `json:"deviceId"`
}

type DeviceMessageV1 struct {
	DeviceId   int64  `json:"deviceId"`
	DeviceName string `json:"deviceName"`
}

type DeviceListResponseV1 struct {
	Devices []DeviceMessageV1 `json:"Devices"`
}

// V2 objects below

type MigrateRequestV2 struct {
	SyncCode   string `json:"syncCode"`
	DeviceId   int64  `json:"deviceId"`
	DeviceName string `json:"deviceName"`
}

type UserDeviceResponseV2 struct {
	UserId     uuid.UUID `json:"userId"`
	DeviceId   uuid.UUID `json:"deviceId"`
	DeviceName string    `json:"deviceName"`
}

type CreateChainRequestV2 struct {
	DeviceName string `json:"deviceName"`
}

type JoinChainRequestV2 struct {
	UserId     uuid.UUID `json:"userId"`
	DeviceName string    `json:"deviceName"`
}
