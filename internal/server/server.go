package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/felixge/httpsnoop"
	"github.com/spacecowboy/feeder-sync/internal/store"
	"github.com/spacecowboy/feeder-sync/internal/store/sqlite"
)

type FeederServer struct {
	store   store.DataStore
	handler http.Handler
}

func NewServer() (*FeederServer, error) {
	store, err := sqlite.New("./sqlite.db")
	if err != nil {
		return nil, err
	}

	if err := store.RunMigrations("file://./migrations"); err != nil {
		return nil, err
	}

	return NewServerWithStore(&store)
}

func NewServerWithStore(dataStore store.DataStore) (*FeederServer, error) {
	server := FeederServer{
		store: dataStore,
	}

	router := http.NewServeMux()

	router.Handle("/api/v2/migrate", http.HandlerFunc(server.handleMigrateV2))
	router.Handle("/api/v1/create", http.HandlerFunc(server.handleCreateV1))
	router.Handle("/api/v2/create", http.HandlerFunc(server.handleCreateV2))
	router.Handle("/api/v1/join", http.HandlerFunc(server.handleJoinV1))
	router.Handle("/api/v2/join", http.HandlerFunc(server.handleJoinV2))
	router.Handle("/api/v1/ereadmark", http.HandlerFunc(server.handleReadmarkV1))
	router.Handle("/api/v1/devices", http.HandlerFunc(server.handleDeviceGetV1))
	// Ending slash is like a wildcard
	router.Handle("/api/v1/devices/", http.HandlerFunc(server.handleDeviceDeleteV1))
	router.Handle("/api/v1/feeds", http.HandlerFunc(server.handleFeedsV1))

	wrappedRouter := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m := httpsnoop.CaptureMetrics(router, w, r)
		log.Printf(
			"%s %s (code=%d dt=%s written=%d)",
			r.Method,
			r.URL,
			m.Code,
			m.Duration,
			m.Written,
		)
	},
	)

	server.handler = wrappedRouter

	return &server, nil
}

func (s *FeederServer) Close() error {
	return s.store.Close()
}

func (s *FeederServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func (s *FeederServer) handleDeviceGetV1(w http.ResponseWriter, r *http.Request) {
	if !s.ensureBasicAuthOrError(w, r) {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
		return
	}

	syncCode := r.Header.Get("X-FEEDER-ID")
	if syncCode == "" {
		log.Println("No sync code in header")
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	legacyDeviceIdString := r.Header.Get("X-FEEDER-DEVICE-ID")
	if legacyDeviceIdString == "" {
		log.Println("No device id in header")
		http.Error(w, "Missing Device ID", http.StatusBadRequest)
		return
	}
	legacyDeviceId, err := strconv.ParseInt(legacyDeviceIdString, 10, 64)
	if err != nil {
		log.Printf("Device Id was not a 64 bit number: %s", legacyDeviceIdString)
		http.Error(w, "Bad Device ID", http.StatusBadRequest)
		return
	}

	userDevice, err := s.store.GetLegacyDevice(syncCode, legacyDeviceId)
	if err != nil {
		log.Printf("Could not find userdevice %d: %s", legacyDeviceId, err.Error())
		if err == store.ErrNoSuchDevice {
			// Used by clients
			http.Error(w, DEVICE_NOT_REGISTERED, http.StatusBadRequest)
			return
		}
		http.Error(w, "Could not fetch device", http.StatusBadRequest)
		return
	}

	_, err = s.store.UpdateLastSeenForDevice(userDevice)
	if err != nil {
		log.Printf("Failed to update last seen for device %s: %s", userDevice.DeviceId, err.Error())
		http.Error(w, "Something bad happened", http.StatusInternalServerError)
		return
	}

	devices, err := s.store.GetDevices(userDevice.UserId)
	if err != nil {
		log.Printf("Failed to fetch devices for user %s: %s", userDevice.UserId, err.Error())
		http.Error(w, "Something bad happened", http.StatusInternalServerError)
		return
	}

	response := DeviceListResponseV1{
		Devices: make([]DeviceMessageV1, 0, len(devices)),
	}

	for _, device := range devices {
		response.Devices = append(
			response.Devices,
			DeviceMessageV1{
				DeviceId:   device.LegacyDeviceId,
				DeviceName: device.DeviceName,
			},
		)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Could not encode devices: %s", err.Error())
		http.Error(w, "Could not encode response", http.StatusInternalServerError)
		return
	}
}

func (s *FeederServer) handleDeviceDeleteV1(w http.ResponseWriter, r *http.Request) {
	if !s.ensureBasicAuthOrError(w, r) {
		return
	}

	if r.Method != "DELETE" {
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
		return
	}

	if path.Dir(r.URL.Path) != "/api/v1/devices" {
		http.Error(w, "No no", http.StatusNotFound)
		return
	}

	syncCode := r.Header.Get("X-FEEDER-ID")
	if syncCode == "" {
		log.Println("No sync code in header")
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	legacyDeviceIdString := r.Header.Get("X-FEEDER-DEVICE-ID")
	if legacyDeviceIdString == "" {
		log.Println("No device id in header")
		http.Error(w, "Missing Device ID", http.StatusBadRequest)
		return
	}
	legacyDeviceId, err := strconv.ParseInt(legacyDeviceIdString, 10, 64)
	if err != nil {
		log.Printf("Device Id was not a 64 bit number: %s", legacyDeviceIdString)
		http.Error(w, "Bad Device ID", http.StatusBadRequest)
		return
	}

	userDevice, err := s.store.GetLegacyDevice(syncCode, legacyDeviceId)
	if err != nil {
		log.Printf("Could not find userdevice %d: %s", legacyDeviceId, err.Error())
		if err == store.ErrNoSuchDevice {
			// Used by clients
			http.Error(w, DEVICE_NOT_REGISTERED, http.StatusBadRequest)
			return
		}
		http.Error(w, "Could not fetch device", http.StatusBadRequest)
		return
	}

	_, err = s.store.UpdateLastSeenForDevice(userDevice)
	if err != nil {
		log.Printf("Failed to update last seen for device %s: %s", userDevice.DeviceId, err.Error())
		http.Error(w, "Something bad happened", http.StatusInternalServerError)
		return
	}

	targetLegacyDeviceIdString := path.Base(r.URL.Path)
	targetLegacyDeviceId, err := strconv.ParseInt(targetLegacyDeviceIdString, 10, 64)
	if err != nil {
		log.Printf("Device Id was not a 64 bit number: %s", legacyDeviceIdString)
		http.Error(w, "Bad Device ID", http.StatusBadRequest)
		return
	}

	_, err = s.store.RemoveDeviceWithLegacy(userDevice.UserDbId, targetLegacyDeviceId)
	if err != nil {
		log.Printf("Failed to delete device %d for device %s: %s", targetLegacyDeviceId, userDevice.DeviceId, err.Error())
		http.Error(w, "Something bad happened", http.StatusInternalServerError)
		return
	}

	devices, err := s.store.GetDevices(userDevice.UserId)
	if err != nil {
		log.Printf("Failed to fetch devices for user %s: %s", userDevice.UserId, err.Error())
		http.Error(w, "Something bad happened", http.StatusInternalServerError)
		return
	}

	response := DeviceListResponseV1{
		Devices: make([]DeviceMessageV1, len(devices)),
	}

	for _, device := range devices {
		response.Devices = append(
			response.Devices,
			DeviceMessageV1{
				DeviceId:   device.LegacyDeviceId,
				DeviceName: device.DeviceName,
			},
		)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Could not encode devices: %s", err.Error())
		http.Error(w, "Could not encode response", http.StatusInternalServerError)
		return
	}
}

func matchesEtag(requestEtag string, etagValue string) bool {
	if requestEtag == "*" || etagValue == "" {
		return true
	}

	if requestEtag == etagValue {
		return true
	}

	etagValueNoPrefix, _ := strings.CutPrefix(etagValue, "W/")

	return requestEtag == etagValueNoPrefix
}

func (s *FeederServer) handleFeedsV1(w http.ResponseWriter, r *http.Request) {
	if !s.ensureBasicAuthOrError(w, r) {
		return
	}

	if r.Method != "GET" && r.Method != "POST" {
		log.Printf("Unsupported method: %s", r.Method)
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
		return
	}

	syncCode := r.Header.Get("X-FEEDER-ID")
	if syncCode == "" {
		log.Println("No sync code in header")
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	legacyDeviceIdString := r.Header.Get("X-FEEDER-DEVICE-ID")
	if legacyDeviceIdString == "" {
		log.Println("No device id in header")
		http.Error(w, "Missing Device ID", http.StatusBadRequest)
		return
	}
	legacyDeviceId, err := strconv.ParseInt(legacyDeviceIdString, 10, 64)
	if err != nil {
		log.Printf("Device Id was not a 64 bit number: %s", legacyDeviceIdString)
		http.Error(w, "Bad Device ID", http.StatusBadRequest)
		return
	}

	userDevice, err := s.store.GetLegacyDevice(syncCode, legacyDeviceId)
	if err != nil {
		log.Printf("Could not find userdevice %d: %s", legacyDeviceId, err.Error())
		if err == store.ErrNoSuchDevice {
			// Used by clients
			http.Error(w, DEVICE_NOT_REGISTERED, http.StatusBadRequest)
			return
		}
		http.Error(w, "Could not fetch device", http.StatusBadRequest)
		return
	}

	_, err = s.store.UpdateLastSeenForDevice(userDevice)
	if err != nil {
		log.Printf("Failed to update last seen for device %s: %s", userDevice.DeviceId, err.Error())
		http.Error(w, "Something bad happened", http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case "GET":
		s.handleGetFeedsV1(userDevice, w, r)
	case "POST":
		s.handlePostFeedsV1(userDevice, w, r)
	}
}

func (s *FeederServer) handleGetFeedsV1(userDevice store.UserDevice, w http.ResponseWriter, r *http.Request) {
	feeds, err := s.store.GetLegacyFeeds(userDevice.UserId)
	if err != nil {
		if err == store.ErrNoFeeds {
			w.WriteHeader(http.StatusNoContent)
			return
		} else {
			log.Printf("GetLegacyFeeds error: %s", err.Error())
			http.Error(w, "Something bad", http.StatusInternalServerError)
			return
		}
	}

	requestEtag := r.Header.Get("If-None-Match")
	if matchesEtag(requestEtag, feeds.Etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	response := GetFeedsResponseV1{
		ContentHash: feeds.ContentHash,
		Encrypted:   feeds.Content,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Could not encode feeds: %s", err.Error())
		http.Error(w, "Could not encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Cache-Control", "private, must-revalidate")
	w.Header().Add("ETag", feeds.Etag)
}

func (s *FeederServer) handlePostFeedsV1(userDevice store.UserDevice, w http.ResponseWriter, r *http.Request) {
	var currentEtag string
	feeds, err := s.store.GetLegacyFeeds(userDevice.UserId)
	if err != nil {
		if err == store.ErrNoFeeds {
			currentEtag = ""
		} else {
			log.Printf("PostLegacyFeeds error: %s", err.Error())
			http.Error(w, "Something bad", http.StatusInternalServerError)
			return
		}
	} else {
		currentEtag = feeds.Etag
	}

	requestEtag := r.Header.Get("If-Match")
	if !matchesEtag(requestEtag, currentEtag) {
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}

	if r.Body == nil {
		log.Println("No body")
		http.Error(w, "No body", http.StatusBadRequest)
		return
	}

	var feedsRequest UpdateFeedsRequestV1

	if err := json.NewDecoder(r.Body).Decode(&feedsRequest); err != nil {
		log.Println("Bad body")
		http.Error(w, "Bad body", http.StatusBadRequest)
		return
	}

	_, err = s.store.UpdateLegacyFeeds(
		userDevice.UserDbId,
		feedsRequest.ContentHash,
		feedsRequest.Encrypted,
		etagValueForInt64(feedsRequest.ContentHash),
	)

	if err != nil {
		log.Printf("Update feeds failed: %s", err.Error())
		http.Error(w, "Something bad", http.StatusInternalServerError)
		return
	}

	response := UpdateFeedsResponseV1{
		ContentHash: feedsRequest.ContentHash,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Could not encode feeds: %s", err.Error())
		http.Error(w, "Could not encode response", http.StatusInternalServerError)
		return
	}
}

func etagValueForInt64(data int64) string {
	return fmt.Sprintf("W/\"%d\"", data)
}

func (s *FeederServer) handleReadmarkV1(w http.ResponseWriter, r *http.Request) {
	if !s.ensureBasicAuthOrError(w, r) {
		return
	}

	syncCode := r.Header.Get("X-FEEDER-ID")
	if syncCode == "" {
		log.Println("No sync code in header")
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	legacyDeviceIdString := r.Header.Get("X-FEEDER-DEVICE-ID")
	if legacyDeviceIdString == "" {
		log.Println("No device id in header")
		http.Error(w, "Missing Device ID", http.StatusBadRequest)
		return
	}
	legacyDeviceId, err := strconv.ParseInt(legacyDeviceIdString, 10, 64)
	if err != nil {
		log.Printf("Device Id was not a 64 bit number: %s", legacyDeviceIdString)
		http.Error(w, "Bad Device ID", http.StatusBadRequest)
		return
	}

	userDevice, err := s.store.GetLegacyDevice(syncCode, legacyDeviceId)
	if err != nil {
		if err == store.ErrNoSuchDevice {
			// Used by clients
			http.Error(w, DEVICE_NOT_REGISTERED, http.StatusBadRequest)
			return
		}
		log.Printf("Could not find userdevice %d: %s", legacyDeviceId, err.Error())
		http.Error(w, "Could not fetch device", http.StatusBadRequest)
		return
	}

	_, err = s.store.UpdateLastSeenForDevice(userDevice)
	if err != nil {
		log.Printf("Failed to update last seen for device %s: %s", userDevice.DeviceId, err.Error())
		http.Error(w, "Something bad happened", http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case "GET":
		response := GetReadmarksResponseV1{
			ReadMarks: make([]ReadMarkV1, 0, 1),
		}

		sinceRaw := r.URL.Query().Get("since")
		// milliseconds
		var since int64 = 0
		if sinceRaw != "" {
			since, err = strconv.ParseInt(sinceRaw, 10, 64)

			if err != nil {
				http.Error(w, "Invalid value for since-queryParam", http.StatusBadRequest)
				return
			}
		}

		articles, err := s.store.GetArticles(userDevice.UserId, since)

		if err != nil {
			log.Printf("Could not fetch articles: %s", err.Error())
			http.Error(w, "Could not fetch articles", http.StatusInternalServerError)
			return
		}

		for _, article := range articles {
			response.ReadMarks = append(
				response.ReadMarks,
				ReadMarkV1{
					Encrypted: article.Identifier,
					Timestamp: article.ReadTime,
				},
			)
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Could not encode articles: %s", err.Error())
			http.Error(w, "Could not encode response", http.StatusInternalServerError)
			return
		}

	case "POST":
		if r.Body == nil {
			log.Println("No body")
			http.Error(w, "No body", http.StatusBadRequest)
			return
		}

		var sendRequest SendReadMarksRequestV1

		if err := json.NewDecoder(r.Body).Decode(&sendRequest); err != nil {
			log.Println("Bad body")
			http.Error(w, "Bad body", http.StatusBadRequest)
			return
		}

		for _, readmark := range sendRequest.ReadMarks {
			if err := s.store.AddLegacyArticle(userDevice.UserDbId, readmark.Encrypted); err != nil {
				log.Printf("Failed to add article: %v", err.Error())
				http.Error(w, "Failed to store article", http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusNoContent)

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

func (s *FeederServer) ensureBasicAuthOrError(w http.ResponseWriter, r *http.Request) bool {
	username, password, ok := r.BasicAuth()
	if !ok {
		http.Error(w, "Missing auth", http.StatusUnauthorized)
		return false
	}

	if username != HARDCODED_USER || password != HARDCODED_PASSWORD {
		http.Error(w, "Bad auth", http.StatusUnauthorized)
		return false
	}

	return true
}

func (s *FeederServer) handleCreateV1(w http.ResponseWriter, r *http.Request) {
	if !s.ensureBasicAuthOrError(w, r) {
		return
	}

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
	if !s.ensureBasicAuthOrError(w, r) {
		return
	}

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
	if !s.ensureBasicAuthOrError(w, r) {
		return
	}

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
		case "user not found":
			http.Error(w, "user not found", http.StatusNotFound)
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
	if !s.ensureBasicAuthOrError(w, r) {
		return
	}

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

// Used by clients
var DEVICE_NOT_REGISTERED = "Device not registered"

// Used internally
var HARDCODED_USER = "feeder_user"
var HARDCODED_PASSWORD = "feeder_secret_1234"
