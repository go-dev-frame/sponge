package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/go-dev-frame/sponge/pkg/gin/middleware/auth"
)

// 1. Universal session middleware example refer to https://github.com/gin-contrib/sessions?tab=readme-ov-file#basic-examples

// -------------------------------------------------------------------------------------------

// 2. Special session for rails

// RailsCookieAuthMiddleware validates and decrypts a Rails encrypted cookie,
// attaches the session payload to context under key "rails_session".
func RailsCookieAuthMiddleware(secretKeyBase string, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(cookieName)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "Missing cookie"})
			return
		}

		session, err := auth.DecodeSignedCookie(secretKeyBase, cookie, cookieName)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid cookie"})
			return
		}

		c.Set("rails_session", session)
		c.Next()
	}
}
