package server

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/felixge/httpsnoop"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/spacecowboy/feeder-sync/internal/db"
	"github.com/spacecowboy/feeder-sync/internal/store"
)

type FeederServer struct {
	queries *db.Queries
	pgConn  *pgx.Conn
	handler http.Handler
}

func NewServerWithPostgres(connString string) (*FeederServer, error) {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return nil, err
	}

	// Migrations
	// ./migrations/...

	return NewServerWithStore(conn)
}

func NewServerWithStore(conn *pgx.Conn) (*FeederServer, error) {
	queries := db.New(conn)
	server := FeederServer{
		queries: queries,
		pgConn:  conn,
	}

	router := http.NewServeMux()

	router.Handle("/health", http.HandlerFunc(server.handleHealth))
	router.Handle("/ready", http.HandlerFunc(server.handleReady))

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
	return s.pgConn.Close(context.Background())
}

func (s *FeederServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func (s *FeederServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if s.handler != nil && s.queries != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server is not initialized"))
	}
}

func (s *FeederServer) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	// Check if the database connection is alive
	if err := s.pgConn.Ping(ctx); err != nil {
		log.Printf("Database connection is not ready: %s", err.Error())
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Database connection is not ready"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready"))
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

func etagValueForInt64(data int64) string {
	return fmt.Sprintf("W/\"%d\"", data)
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

// Used by clients
var DEVICE_NOT_REGISTERED = "Device not registered"

// Used internally
var HARDCODED_USER = "feeder_user"
var HARDCODED_PASSWORD = "feeder_secret_1234"

func (s *FeederServer) handleDeviceGetV1(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	etagBytes, err := s.queries.GetLegacyDevicesEtag(ctx, syncCode)
	if err != nil {
		if err == store.ErrNoSuchDevice {
			http.Error(w, DEVICE_NOT_REGISTERED, http.StatusBadRequest)
			return
		} else {
			log.Printf("GetLegacyDevicesEtag error: %s", err.Error())
			http.Error(w, "Something bad", http.StatusInternalServerError)
			return
		}
	}
	etag := string(etagBytes)
	requestEtag := r.Header.Get("If-None-Match")
	if matchesEtag(requestEtag, etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	legacyDeviceRow, err := s.queries.GetLegacyDevice(
		ctx,
		db.GetLegacyDeviceParams{
			LegacySyncCode: syncCode,
			LegacyDeviceID: legacyDeviceId,
		},
	)
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

	err = s.queries.UpdateLastSeenForDevice(
		ctx,
		db.UpdateLastSeenForDeviceParams{
			LastSeen: pgtype.Timestamptz{
				Time:  time.Now(),
				Valid: true,
			},
			DbID: legacyDeviceRow.Device.DbID,
		},
	)
	if err != nil {
		log.Printf("Failed to update last seen for device %s: %s", legacyDeviceRow.Device.DeviceID, err.Error())
		http.Error(w, "Something bad happened", http.StatusInternalServerError)
		return
	}

	devices, err := s.queries.GetDevices(ctx, legacyDeviceRow.User.UserID)
	if err != nil {
		log.Printf("Failed to fetch devices for user %s: %s", legacyDeviceRow.User.UserID, err.Error())
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
				DeviceId:   device.Device.LegacyDeviceID,
				DeviceName: device.Device.DeviceName,
			},
		)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Could not encode devices: %s", err.Error())
		http.Error(w, "Could not encode response", http.StatusInternalServerError)
		return
	}

	if etag != "" {
		w.Header().Add("Cache-Control", "private, must-revalidate")
		w.Header().Add("ETag", etag)
	}
}

func (s *FeederServer) handleDeviceDeleteV1(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	legacyDeviceRow, err := s.queries.GetLegacyDevice(
		ctx,
		db.GetLegacyDeviceParams{
			LegacySyncCode: syncCode,
			LegacyDeviceID: legacyDeviceId,
		},
	)
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

	err = s.queries.UpdateLastSeenForDevice(
		ctx,
		db.UpdateLastSeenForDeviceParams{
			LastSeen: pgtype.Timestamptz{
				Time:  time.Now(),
				Valid: true,
			},
			DbID: legacyDeviceRow.Device.DbID,
		},
	)
	if err != nil {
		log.Printf("Failed to update last seen for device %s: %s", legacyDeviceRow.Device.DeviceID, err.Error())
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

	err = s.queries.DeleteDeviceWithLegacyId(
		ctx,
		db.DeleteDeviceWithLegacyIdParams{
			UserDbID:       legacyDeviceRow.Device.UserDbID,
			LegacyDeviceID: legacyDeviceRow.Device.LegacyDeviceID,
		},
	)
	if err != nil {
		log.Printf("Failed to delete device %d for device %s: %s", targetLegacyDeviceId, legacyDeviceRow.Device.DeviceID, err.Error())
		http.Error(w, "Something bad happened", http.StatusInternalServerError)
		return
	}

	devices, err := s.queries.GetDevices(ctx, legacyDeviceRow.User.UserID)
	if err != nil {
		log.Printf("Failed to fetch devices for user %s: %s", legacyDeviceRow.User.UserID, err.Error())
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
				DeviceId:   device.Device.LegacyDeviceID,
				DeviceName: device.Device.DeviceName,
			},
		)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Could not encode devices: %s", err.Error())
		http.Error(w, "Could not encode response", http.StatusInternalServerError)
		return
	}
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

	ctx := r.Context()

	userDevice, err := s.queries.GetLegacyDevice(
		ctx,
		db.GetLegacyDeviceParams{
			LegacySyncCode: syncCode,
			LegacyDeviceID: legacyDeviceId,
		},
	)
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

	err = s.queries.UpdateLastSeenForDevice(
		ctx,
		db.UpdateLastSeenForDeviceParams{
			DbID: userDevice.Device.DbID,
			LastSeen: pgtype.Timestamptz{
				Time:  time.Now(),
				Valid: true,
			},
		},
	)
	if err != nil {
		log.Printf("Failed to update last seen for device %s: %s", userDevice.Device.DeviceID, err.Error())
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

func (s *FeederServer) handleGetFeedsV1(userDevice db.GetLegacyDeviceRow, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	feedsEtag, err := s.queries.GetLegacyFeedsEtag(ctx, userDevice.User.UserID)
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
	if matchesEtag(requestEtag, feedsEtag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	feeds, err := s.queries.GetLegacyFeeds(ctx, userDevice.User.UserID)
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

	response := GetFeedsResponseV1{
		ContentHash: feeds.ContentHash,
		Encrypted:   feeds.Content,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Could not encode feeds: %s", err.Error())
		http.Error(w, "Could not encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Cache-Control", "private, must-revalidate")
	w.Header().Add("ETag", feeds.Etag)
}

func (s *FeederServer) handlePostFeedsV1(userDevice db.GetLegacyDeviceRow, w http.ResponseWriter, r *http.Request) {
	var currentEtag string
	ctx := r.Context()
	feeds, err := s.queries.GetLegacyFeeds(ctx, userDevice.User.UserID)
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

	_, err = s.queries.UpdateLegacyFeeds(
		ctx,
		db.UpdateLegacyFeedsParams{
			UserDbID:    userDevice.User.DbID,
			ContentHash: feedsRequest.ContentHash,
			Content:     feedsRequest.Encrypted,
			Etag:        etagValueForInt64(feedsRequest.ContentHash),
		},
	)

	if err != nil {
		log.Printf("Update feeds failed: %s", err.Error())
		http.Error(w, "Something bad", http.StatusInternalServerError)
		return
	}

	response := UpdateFeedsResponseV1{
		ContentHash: feedsRequest.ContentHash,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Could not encode feeds: %s", err.Error())
		http.Error(w, "Could not encode response", http.StatusInternalServerError)
		return
	}
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

	ctx := r.Context()
	userDevice, err := s.queries.GetLegacyDevice(
		ctx,
		db.GetLegacyDeviceParams{
			LegacySyncCode: syncCode,
			LegacyDeviceID: legacyDeviceId,
		},
	)
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

	err = s.queries.UpdateLastSeenForDevice(
		ctx,
		db.UpdateLastSeenForDeviceParams{
			DbID: userDevice.Device.DbID,
			LastSeen: pgtype.Timestamptz{
				Time:  time.Now(),
				Valid: true,
			},
		},
	)
	if err != nil {
		log.Printf("Failed to update last seen for device %s: %s", userDevice.Device.DeviceID, err.Error())
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

		ctx := r.Context()
		articles, err := s.queries.GetArticles(
			ctx,
			db.GetArticlesParams{
				UserID: userDevice.User.UserID,
				UpdatedAt: pgtype.Timestamptz{
					Time:  time.UnixMilli(since),
					Valid: true,
				},
			},
		)

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
					Timestamp: article.ReadTime.Time.UnixMilli(),
				},
			)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
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

		timestamp := pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		}

		// TODO: Investigate COPY protocol
		for _, readmark := range sendRequest.ReadMarks {
			if _, err := s.queries.InsertArticle(
				ctx,
				db.InsertArticleParams{
					UserDbID:   userDevice.User.DbID,
					Identifier: readmark.Encrypted,
					ReadTime:   timestamp,
					UpdatedAt:  timestamp,
				},
			); err != nil {
				log.Printf("Failed to add article: %v", err.Error())
				http.Error(w, "Failed to store article", http.StatusInternalServerError)
				return
			}
			/*
						if err != nil {
					if !strings.Contains(err.Error(), "UNIQUE constraint failed: articles.user_db_id, articles.identifier") {
						return err
					}
				}
			*/
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not supported", http.StatusBadRequest)
		return
	}
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

	/*
		userDevice := store.UserDevice{
				UserId:         uuid.New(),
				DeviceId:       uuid.New(),
				DeviceName:     deviceName,
				LegacyDeviceId: rand.Int63(),
				LastSeen:       time.Now().UnixMilli(),
			}

			legacySyncCode, err := randomLegacySyncCode()

			if err != nil {
				log.Printf("could not generate sync code: %s", err.Error())
				return userDevice, err
			}

			userDevice.LegacySyncCode = legacySyncCode

			// Insert user
			err = s.Db.QueryRow(
				"INSERT INTO users (user_id, legacy_sync_code) VALUES ($1, $2) RETURNING db_id",
				userDevice.UserId,
				userDevice.LegacySyncCode,
			).Scan(&userDevice.UserDbId)
			if err != nil {
				log.Printf("could not insert user: %s", err.Error())
				return userDevice, err
			}

			return s.AddDeviceToUser(userDevice) // this is InsertDevice
	*/

	userDevice, err := s.RegisterNewUser(r.Context(), createChainRequest.DeviceName)
	if err != nil {
		http.Error(w, "Badness", http.StatusInternalServerError)
		return
	}

	response := JoinChainResponseV1{
		SyncCode: userDevice.User.LegacySyncCode,
		DeviceId: userDevice.Device.DbID,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Could not encode JoinChainResponseV1", http.StatusInternalServerError)
		return
	}
}

type UserAndDevice struct {
	User   db.User
	Device db.Device
}

func (s *FeederServer) RegisterNewUser(ctx context.Context, deviceName string) (UserAndDevice, error) {
	// userDevice := store.UserDevice{
	// 	UserId:         uuid.New(),
	// 	DeviceId:       uuid.New(),
	// 	DeviceName:     deviceName,
	// 	LegacyDeviceId: rand.Int63(),
	// 	LastSeen:       time.Now().UnixMilli(),
	// }
	var result UserAndDevice

	legacySyncCode, err := randomLegacySyncCode()

	if err != nil {
		log.Printf("could not generate sync code: %s", err.Error())
		return result, err
	}

	// Insert user
	insertUserParams := db.InsertUserParams{
		UserID:         uuid.NewString(),
		LegacySyncCode: legacySyncCode,
	}
	user, err := s.queries.InsertUser(
		ctx,
		insertUserParams,
	)
	if err != nil {
		log.Printf("could not insert user: %s", err.Error())
		return result, err
	}

	result.User = user

	device, err := s.queries.InsertDevice(
		ctx,
		db.InsertDeviceParams{
			UserDbID:       user.DbID,
			DeviceID:       uuid.NewString(),
			DeviceName:     deviceName,
			LegacyDeviceID: rand.Int63(),
			LastSeen: pgtype.Timestamptz{
				Time:  time.Now(),
				Valid: true,
			},
		},
	)
	if err != nil {
		log.Printf("could not insert device: %s", err.Error())
		return result, err
	}

	result.Device = device

	return result, nil
}

func (s *FeederServer) AddDeviceToChainWithLegacy(ctx context.Context, syncCode string, deviceName string) (UserAndDevice, error) {
	// Fetch user with legacy sync code
	user, err := s.queries.GetUserBySyncCode(ctx, syncCode)
	if err != nil {
		return UserAndDevice{}, err
	}

	return s.AddDeviceToUser(ctx, user, deviceName)
}

func (s *FeederServer) AddDeviceToChain(ctx context.Context, userId string, deviceName string) (UserAndDevice, error) {
	user, err := s.queries.GetUserByUserId(ctx, userId)
	if err != nil {
		return UserAndDevice{}, err
	}

	return s.AddDeviceToUser(ctx, user, deviceName)
}

func (s *FeederServer) AddDeviceToUser(ctx context.Context, user db.User, deviceName string) (UserAndDevice, error) {
	var result UserAndDevice
	device, err := s.queries.InsertDevice(
		ctx,
		db.InsertDeviceParams{
			DeviceID:   uuid.NewString(),
			DeviceName: deviceName,
			LastSeen: pgtype.Timestamptz{
				Time:  time.Now(),
				Valid: true,
			},
			LegacyDeviceID: rand.Int63(),
			UserDbID:       user.DbID,
		},
	)
	if err != nil {
		log.Printf("could not insert device: %s", err.Error())
		return result, err
	}
	result.Device = device
	result.User = user
	return result, nil
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

	userDevice, err := s.AddDeviceToChainWithLegacy(r.Context(), syncCode, joinChainRequest.DeviceName)
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
		SyncCode: userDevice.User.LegacySyncCode,
		DeviceId: userDevice.Device.LegacyDeviceID,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
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

	ctx := r.Context()
	userDevice, err := s.RegisterNewUser(ctx, createChainRequest.DeviceName)
	if err != nil {
		http.Error(w, "Badness", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// TODO will explode with changed type of userDevice
	if err := json.NewEncoder(w).Encode(userDevice); err != nil {
		http.Error(w, "Could not encode UserDevice", http.StatusInternalServerError)
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

	userDevice, err := s.AddDeviceToChain(r.Context(), joinChainRequest.UserId.String(), joinChainRequest.DeviceName)
	if err != nil {
		switch err.Error() {
		case "No such user":
			http.Error(w, "No such chain", http.StatusNotFound)
		default:
			http.Error(w, "Badness", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(userDevice); err != nil {
		http.Error(w, "Could not encode UserDevice", http.StatusInternalServerError)
		return
	}
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
