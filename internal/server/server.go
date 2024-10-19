package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/spacecowboy/feeder-sync/build/gen/db"
	"github.com/spacecowboy/feeder-sync/internal/middleware"
	"github.com/spacecowboy/feeder-sync/internal/repository"
)

type FeederServer struct {
	repo   repository.Repository
	Router *gin.Engine
}

func NewServerWithPostgres(connString string) (*FeederServer, error) {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return nil, err
	}

	repo := repository.NewPostgresRepository(conn)

	return NewServerWithRepo(repo)
}

func NewServerWithRepo(repo repository.Repository) (*FeederServer, error) {
	router := gin.Default()

	server := FeederServer{
		repo:   repo,
		Router: router,
	}

	// Middleware
	assertBasicAuth := middleware.AssertBasicAuth()
	assertUser := middleware.AssertRegisteredUser(repo)
	assertDevice := middleware.AssertRegisteredDevice(repo)
	updateLastSeen := middleware.UpdateLastSeenForDevice(repo)

	// These have no middleware
	router.GET("/health", server.handleHealth)
	router.GET("/ready", server.handleReady)

	// Create only checks auth
	apiKeyOnly := router.Group("/api", assertBasicAuth)
	{
		apiKeyOnly.POST("v1/create", server.handleCreateV1)
		apiKeyOnly.POST("v2/create", server.handleCreateV2)
	}

	// auth and UserID
	apiKeyUserId := router.Group("/api", assertBasicAuth, assertUser)
	{
		apiKeyUserId.POST("v1/join", server.handleJoinV1)
		apiKeyUserId.POST("v2/join", server.handleJoinV2)
	}

	// auth, userid, deviceid
	fullyAuthed := router.Group("/api", assertBasicAuth, assertUser, assertDevice, updateLastSeen)
	{
		fullyAuthed.GET("v1/ereadmark", server.handleGETReadmarkV1)
		fullyAuthed.POST("v1/ereadmark", server.handlePOSTReadmarkV1)
		fullyAuthed.GET("v1/devices", server.handleDeviceGetV1)
		fullyAuthed.DELETE("v1/devices/:id", server.handleDeviceDeleteV1)
		fullyAuthed.GET("v1/feeds", server.handleGETFeedsV1)
		fullyAuthed.POST("v1/feeds", server.handlePOSTFeedsV1)
	}

	// wrappedRouter := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	m := httpsnoop.CaptureMetrics(router, w, r)
	// 	log.Printf(
	// 		"%s %s (code=%d dt=%s written=%d)",
	// 		r.Method,
	// 		r.URL,
	// 		m.Code,
	// 		m.Duration,
	// 		m.Written,
	// 	)
	// },
	// )

	// server.handler = wrappedRouter

	return &server, nil
}

func (s *FeederServer) Close() error {
	return s.repo.Close(context.Background())
}

func (s *FeederServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Router.ServeHTTP(w, r)
}

func (s *FeederServer) handleHealth(c *gin.Context) {
	if s.Router != nil && s.repo != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": "Initialized",
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "Server is not initialized",
		})
	}
}

func (s *FeederServer) handleReady(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, 5*time.Second)
	defer cancel()
	// Check if the database connection is alive
	if err := s.repo.PingContext(ctx); err != nil {
		log.Printf("Database connection is not ready: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"status": "Database connection is not ready",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "Ready",
	})
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

func (s *FeederServer) handleDeviceGetV1(c *gin.Context) {
	user := c.MustGet("user").(db.User)

	etag, err := s.repo.GetDevicesEtag(c, user)
	if err != nil {
		log.Printf("GetLegacyDevicesEtag error: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Something bad"})
		return
	}

	requestEtag := c.GetHeader("If-None-Match")
	if matchesEtag(requestEtag, etag) {
		c.Status(http.StatusNotModified)
		return
	}

	devices, err := s.repo.GetDevices(c, user)
	if err != nil {
		log.Printf("Failed to fetch devices for user %s: %s", user.UserID, err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Something bad"})
		return
	}

	response := DeviceListResponseV1{
		Devices: make([]DeviceMessageV1, 0, len(devices)),
	}

	for _, device := range devices {
		response.Devices = append(
			response.Devices,
			DeviceMessageV1{
				DeviceId:   device.LegacyDeviceID,
				DeviceName: device.DeviceName,
			},
		)
	}

	c.Header("Cache-Control", "private, must-revalidate")
	log.Printf("Setting ETag: %s", etag)
	c.Header("ETag", etag)
	c.JSON(http.StatusOK, response)
}

func (s *FeederServer) handleDeviceDeleteV1(c *gin.Context) {
	user := c.MustGet("user").(db.User)

	legacyDeviceIdString := c.Param("id")
	if legacyDeviceIdString == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Bad Device ID"})
		return
	}

	legacyDeviceId, err := strconv.ParseInt(legacyDeviceIdString, 10, 64)
	if err != nil {
		log.Printf("Device Id was not a 64 bit number: %s", legacyDeviceIdString)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Bad Device ID"})
		return
	}

	_, err = s.repo.RemoveDeviceWithLegacyId(c, user, legacyDeviceId)
	if err != nil {
		log.Printf("Failed to delete device %d: %s", legacyDeviceId, err.Error())
		if err == repository.ErrNoSuchDevice {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "No such device"})
		} else {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Something bad"})
		}
		return
	}

	devices, err := s.repo.GetDevices(c, user)
	if err != nil {
		log.Printf("Failed to fetch devices for user %s: %s", user.UserID, err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Something bad"})
		return
	}

	response := DeviceListResponseV1{
		Devices: make([]DeviceMessageV1, 0, len(devices)),
	}

	for _, device := range devices {
		response.Devices = append(
			response.Devices,
			DeviceMessageV1{
				DeviceId:   device.LegacyDeviceID,
				DeviceName: device.DeviceName,
			},
		)
	}

	c.JSON(http.StatusOK, response)
}

func (s *FeederServer) handleGETFeedsV1(c *gin.Context) {
	user := c.MustGet("user").(db.User)

	feeds, err := s.repo.GetLegacyFeeds(c, user)
	if err != nil {
		if err == repository.ErrNoFeeds {
			c.Status(http.StatusNoContent)
			return
		} else {
			log.Printf("GetLegacyFeeds error: %s", err.Error())
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Something bad"})
			return
		}
	}

	requestEtag := c.GetHeader("If-None-Match")
	if matchesEtag(requestEtag, feeds.Etag) {
		c.Status(http.StatusNotModified)
		return
	}

	response := GetFeedsResponseV1{
		ContentHash: feeds.ContentHash,
		Encrypted:   feeds.Content,
	}

	c.JSON(http.StatusOK, response)
	c.Header("Cache-Control", "private, must-revalidate")
	c.Header("ETag", feeds.Etag)
}

func (s *FeederServer) handlePOSTFeedsV1(c *gin.Context) {
	user := c.MustGet("user").(db.User)

	var currentEtag string

	feeds, err := s.repo.GetLegacyFeeds(c, user)
	if err != nil && err != repository.ErrNoFeeds {
		log.Printf("PostLegacyFeeds error: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Something bad", "err": err.Error()})
		return
	}

	currentEtag = feeds.Etag

	requestEtag := c.GetHeader("If-Match")
	if !matchesEtag(requestEtag, currentEtag) {
		log.Printf("Etag mismatch: [%s] != [%s]", requestEtag, currentEtag)
		c.AbortWithStatus(http.StatusPreconditionFailed)
		return
	}

	var feedsRequest UpdateFeedsRequestV1
	if err := c.BindJSON(&feedsRequest); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if feedsRequest.ContentHash == 0 || feedsRequest.Encrypted == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	_, err = s.repo.UpdateLegacyFeeds(
		c,
		user,
		feedsRequest.ContentHash,
		feedsRequest.Encrypted,
		etagValueForInt64(feedsRequest.ContentHash),
	)

	if err != nil {
		log.Printf("Update feeds failed: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Something bad"})
		return
	}

	response := UpdateFeedsResponseV1{
		ContentHash: feedsRequest.ContentHash,
	}

	c.JSON(http.StatusOK, response)
}

func (s *FeederServer) handleGETReadmarkV1(c *gin.Context) {
	user := c.MustGet("user").(db.User)

	sinceRaw := c.Query("since")
	// milliseconds
	var since int64 = 0
	var err error
	if sinceRaw != "" {
		since, err = strconv.ParseInt(sinceRaw, 10, 64)

		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid value for since-queryParam"})
			return
		}
	}

	articles, err := s.repo.GetArticlesUpdatedSince(c, user, since)

	if err != nil {
		if err == repository.ErrNoReadMarks {
			c.Status(http.StatusNoContent)
			return
		} else {
			log.Printf("Could not fetch articles: %s", err.Error())
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch articles"})
			return
		}
	}

	response := GetReadmarksResponseV1{
		ReadMarks: make([]ReadMarkV1, 0, len(articles)),
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

	c.JSON(http.StatusOK, response)
}

func (s *FeederServer) handlePOSTReadmarkV1(c *gin.Context) {
	user := c.MustGet("user").(db.User)

	var sendRequest SendReadMarksRequestV1

	if err := c.BindJSON(&sendRequest); err != nil {
		log.Println("Bad body")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Bad body"})
		return
	}

	if len(sendRequest.ReadMarks) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "No readmarks"})
		return
	}

	// TODO: Investigate COPY protocol
	for _, readmark := range sendRequest.ReadMarks {
		if _, err := s.repo.AddArticle(c, user, readmark.Encrypted); err != nil {
			log.Printf("Failed to add article: %v", err.Error())
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to store article"})
			return
		}
	}

	c.Status(http.StatusNoContent)
}

func (s *FeederServer) handleCreateV1(c *gin.Context) {
	var createChainRequest CreateChainRequestV1

	if err := c.BindJSON(&createChainRequest); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Bad body"})
		return
	}

	if createChainRequest.DeviceName == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing deviceName"})
		return
	}

	userDevice, err := s.repo.RegisterNewUser(c, createChainRequest.DeviceName)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Badness"})
		return
	}

	response := JoinChainResponseV1{
		SyncCode: userDevice.User.LegacySyncCode,
		DeviceId: userDevice.Device.LegacyDeviceID,
	}

	c.JSON(http.StatusOK, response)
}

func (s *FeederServer) handleJoinV1(c *gin.Context) {
	user := c.MustGet("user").(db.User)

	var joinChainRequest JoinChainRequestV1

	if err := c.BindJSON(&joinChainRequest); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Bad body"})
		return
	}

	if joinChainRequest.DeviceName == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing deviceName"})
		return
	}

	device, err := s.repo.AddDeviceToUser(c, user, joinChainRequest.DeviceName)
	if err != nil {
		switch err.Error() {
		case "user not found":
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "user not found"})
		default:
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Badness"})
		}
		return
	}

	response := JoinChainResponseV1{
		SyncCode: user.LegacySyncCode,
		DeviceId: device.LegacyDeviceID,
	}

	c.JSON(http.StatusOK, response)
}

func (s *FeederServer) handleCreateV2(c *gin.Context) {
	var createChainRequest CreateChainRequestV2

	if err := c.BindJSON(&createChainRequest); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Bad body"})
		return
	}

	if createChainRequest.DeviceName == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing deviceName"})
		return
	}

	userDevice, err := s.repo.RegisterNewUser(c, createChainRequest.DeviceName)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Badness"})
		return
	}

	userId, err := uuid.Parse(userDevice.User.UserID)
	if err != nil {
		log.Printf("Could not parse UUID: %s", userDevice.User.UserID)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Badness"})
		return
	}

	deviceId, err := uuid.Parse(userDevice.Device.DeviceID)
	if err != nil {
		log.Printf("Could not parse UUID: %s", userDevice.Device.DeviceID)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Badness"})
		return
	}

	response := UserDeviceResponseV2{
		UserId:     userId,
		DeviceId:   deviceId,
		DeviceName: userDevice.Device.DeviceName,
	}

	c.JSON(http.StatusOK, response)
}

func (s *FeederServer) handleJoinV2(c *gin.Context) {
	user := c.MustGet("user").(db.User)

	var joinChainRequest JoinChainRequestV2

	if err := c.BindJSON(&joinChainRequest); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Bad body"})
		return
	}

	if joinChainRequest.DeviceName == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing deviceName"})
		return
	}

	device, err := s.repo.AddDeviceToUser(c, user, joinChainRequest.DeviceName)
	if err != nil {
		switch err.Error() {
		case "No such user":
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "No such user"})
		default:
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Badness"})
		}
		return
	}

	userId, err := uuid.Parse(user.UserID)
	if err != nil {
		log.Printf("Could not parse UUID: %s", user.UserID)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Badness"})
		return
	}

	deviceId, err := uuid.Parse(device.DeviceID)
	if err != nil {
		log.Printf("Could not parse UUID: %s", device.DeviceID)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Badness"})
		return
	}

	response := UserDeviceResponseV2{
		UserId:     userId,
		DeviceId:   deviceId,
		DeviceName: device.DeviceName,
	}

	c.JSON(http.StatusOK, response)
}
