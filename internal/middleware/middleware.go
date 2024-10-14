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
			c.AbortWithStatus(http.StatusUnauthorized)
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

func AssertRegisteredDevice(repo repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// This is the legacy device id (int64)
		legacyDeviceIdString := c.GetHeader("X-FEEDER-DEVICE-ID")

		if legacyDeviceIdString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		legacyDeviceId, err := strconv.ParseInt(legacyDeviceIdString, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Unauthorized"})
			return
		}

		user := c.MustGet("user").(db.User)
		device, err := repo.GetDeviceWithLegacyId(c, user, legacyDeviceId)

		if err != nil {
			c.String(http.StatusUnauthorized, DEVICE_NOT_REGISTERED)
			c.Abort()
			return
		}

		c.Set("device", device)
		c.Next()
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
