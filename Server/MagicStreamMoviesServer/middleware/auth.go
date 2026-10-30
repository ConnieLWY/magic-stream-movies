package middleware

import (
	"net/http"
	"slices"
	"strings"

	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/tokens"
	"github.com/gin-gonic/gin"
)

// Context keys for the caller's identity. They are private to this package; other packages use the accessors below.
const (
	ctxUserID = "userID"
	ctxRole   = "role"
)

// Auth validates "Authorization: Bearer <access token>" and stores the user ID and role in the context.
func Auth(tm *tokens.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, found := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !found || tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		claims, err := tm.ValidateAccess(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set(ctxUserID, claims.UserID)
		c.Set(ctxRole, claims.Role)
		c.Next()
	}
}

// RequireRole must run after Auth. A signed-in user with the wrong role gets 403, not 401:
// a 401 would make the frontend think the token expired and try to refresh it.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !slices.Contains(roles, Role(c)) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}
		c.Next()
	}
}

// UserID returns the user ID set by Auth, or "" if Auth did not run.
func UserID(c *gin.Context) string { return c.GetString(ctxUserID) }

// Role returns the role set by Auth, or "" if Auth did not run.
func Role(c *gin.Context) string { return c.GetString(ctxRole) }
