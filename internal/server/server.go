package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/spacecowboy/feeder-sync/internal/store"
	"github.com/spacecowboy/feeder-sync/internal/store/sqlite"
)

type FeederServer struct {
	store  store.DataStore
	router *http.ServeMux
}

func NewServer() (FeederServer, error) {
	store, err := sqlite.New("./sqlite.db")
	if err != nil {
		return FeederServer{}, err
	}

	if err := store.RunMigrations("file://./migrations"); err != nil {
		return FeederServer{}, err
	}

	return NewServerWithStore(store)
}

func NewServerWithStore(store store.DataStore) (FeederServer, error) {
	server := FeederServer{
		store:  store,
		router: http.NewServeMux(),
	}

	server.router.Handle("/api/v2/migrate", http.HandlerFunc(server.handleMigrateV2))
	server.router.Handle("/api/v1/create", http.HandlerFunc(server.handleCreateV1))
	server.router.Handle("/api/v2/create", http.HandlerFunc(server.handleCreateV2))
	server.router.Handle("/api/v1/join", http.HandlerFunc(server.handleJoinV1))
	server.router.Handle("/api/v2/join", http.HandlerFunc(server.handleJoinV2))
	server.router.Handle("/api/v1/ereadmark", http.HandlerFunc(server.handleReadmarkV1))
	// devices
	// feeds

	return server, nil
}

func (s *FeederServer) Close() error {
	return s.store.Close()
}

func (s *FeederServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL)
	s.router.ServeHTTP(w, r)
}

func (s *FeederServer) handleReadmarkV1(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		response := GetReadmarksResponseV1{
			ReadMarks: []ReadMarkV1{
				{Encrypted: "foo"},
				{Encrypted: "foo"},
			},
		}
		store.DataStore.

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Could not encode response", http.StatusInternalServerError)
			return
		}
	case "POST":
		if r.Body == nil {
			http.Error(w, "No body", http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "Method not supported", http.StatusBadRequest)
		return
	}
}

func (s *FeederServer) handleMigrateV2(w http.ResponseWriter, r *http.Request) {
	// Migration is only accepted from the old sync server
	cfWorker := r.Header["Cf-Worker"]
	if cfWorker == nil || len(cfWorker) == 0 || cfWorker[0] != "nononsenseapps.com" {
		http.Error(w, "You bad bad man. Go way.", http.StatusBadRequest)
		return
	}

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
