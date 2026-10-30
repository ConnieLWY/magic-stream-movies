// Package controllers is the HTTP layer: it parses requests, calls services, and turns results and errors into HTTP responses.
// It holds no business rules and never touches the database.
package controllers

import (
	"errors"
	"log"
	"net/http"

	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/services"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/validation"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Business errors mapped to HTTP status codes. Any error not listed here is a 500.
var errorStatus = map[error]int{
	services.ErrInvalidCredentials:  http.StatusUnauthorized,
	services.ErrInvalidRefreshToken: http.StatusUnauthorized,
	services.ErrEmailTaken:          http.StatusConflict,
	services.ErrMovieExists:         http.StatusConflict,
	services.ErrMovieNotFound:       http.StatusNotFound,
	services.ErrPasswordTooLong:     http.StatusBadRequest,
	services.ErrUnknownGenre:        http.StatusBadRequest,
	services.ErrUnrecognizedRanking: http.StatusBadGateway,
}

// respondError writes an error response in the shared format: {"error": "..."}.
// Internal errors are logged in full; the client only sees a generic message.
func respondError(c *gin.Context, err error) {
	for target, status := range errorStatus {
		if errors.Is(err, target) {
			c.JSON(status, gin.H{"error": target.Error()})
			return
		}
	}
	log.Printf("internal error on %s %s: %v", c.Request.Method, c.FullPath(), err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}

// bindJSON parses and validates the request body. On failure it has already written a 400 response and the caller just returns.
func bindJSON(c *gin.Context, v *validator.Validate, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return false
	}
	if err := v.Struct(dst); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed", "details": validation.Describe(err)})
		return false
	}
	return true
}
