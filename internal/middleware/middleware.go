package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spacecowboy/feeder-sync/build/gen/db"
	"github.com/spacecowboy/feeder-sync/internal/repository"
)

// Gin gonic middleware in this file

const (
	HARDCODED_USER     = "feeder_user"
	HARDCODED_PASSWORD = "feeder_secret_1234"
	// Used by clients
	DEVICE_NOT_REGISTERED = "Device not registered"
)

func AssertBasicAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, password, ok := c.Request.BasicAuth()
		if !ok || user != HARDCODED_USER || password != HARDCODED_PASSWORD {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		c.Next()
	}
}

func AssertRegisteredUser(repo repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		syncCode := c.GetHeader("X-FEEDER-ID")
		userIdString := c.GetHeader("X-FEEDER-USER-ID")

		if userIdString != "" {
			userId, err := uuid.Parse(userIdString)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Unauthorized"})
				return
			}

			user, err := repo.GetUserByUserId(c, userId)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
				return
			}

			c.Set("user", user)
		} else if syncCode != "" {
			user, err := repo.GetUserBySyncCode(c, syncCode)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
				return
			}

			c.Set("user", user)
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		c.Next()
	}
}

func AssertRegisteredUserAndDevice(repo repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Legacy user id
		syncCode := c.GetHeader("X-FEEDER-ID")
		// This is the legacy device id (int64)
		deviceIdString := c.GetHeader("X-FEEDER-DEVICE-ID")

		if deviceIdString != "" && syncCode != "" {
			user, device, err := assertLegacyDevice(c, repo, syncCode, deviceIdString)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": DEVICE_NOT_REGISTERED, "value": deviceIdString})
				return
			}
			c.Set("user", user)
			c.Set("device", device)
			c.Next()
			return
		}

		userIdString := c.GetHeader("X-FEEDER-USER-ID")
		if userIdString != "" && deviceIdString != "" {
			user, device, err := assertUserAndDevice(c, repo, userIdString, deviceIdString)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": DEVICE_NOT_REGISTERED, "value": deviceIdString})
				return
			}
			c.Set("user", user)
			c.Set("device", device)
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
}

func UpdateLastSeenForDevice(repo repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		device := c.MustGet("device").(db.Device)

		if err := repo.UpdateLastSeenForDevice(c, device); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}
		c.Next()
	}
}

func assertLegacyDevice(
	c *gin.Context,
	repo repository.Repository,
	syncCode string,
	legacyDeviceIdString string,
) (db.User, db.Device, error) {
	legacyDeviceId, err := strconv.ParseInt(legacyDeviceIdString, 10, 64)
	if err != nil {
		return db.User{}, db.Device{}, err
	}

	result, err := repo.GetUserAndDeviceWithLegacy(c, syncCode, legacyDeviceId)
	if err != nil {
		return db.User{}, db.Device{}, err
	}

	return result.User, result.Device, nil
}

func assertUserAndDevice(
	c *gin.Context,
	repo repository.Repository,
	userIdString string,
	deviceIdString string,
) (db.User, db.Device, error) {
	userId, err := uuid.Parse(userIdString)
	if err != nil {
		return db.User{}, db.Device{}, err
	}

	deviceId, err := uuid.Parse(deviceIdString)
	if err != nil {
		return db.User{}, db.Device{}, err
	}

	result, err := repo.GetUserAndDevice(c, userId, deviceId)
	if err != nil {
		return db.User{}, db.Device{}, err
	}

	return result.User, result.Device, nil
}
