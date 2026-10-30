package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/models"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/services"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/tokens"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

const (
	refreshCookieName = "refresh_token"
	// The refresh token cookie is only sent with /auth/... requests; no other endpoint ever sees it.
	refreshCookiePath = "/auth"
)

type CookieOptions struct {
	Secure   bool
	SameSite http.SameSite
}

type AuthController struct {
	auth     *services.AuthService
	validate *validator.Validate
	cookie   CookieOptions
	timeout  time.Duration
}

func NewAuthController(auth *services.AuthService, v *validator.Validate, cookie CookieOptions, timeout time.Duration) *AuthController {
	return &AuthController{auth: auth, validate: v, cookie: cookie, timeout: timeout}
}

// Register handles POST /auth/register. A successful registration also signs the user in and returns the same body as Login.
func (h *AuthController) Register(c *gin.Context) {
	var req models.UserRegister
	if !bindJSON(c, h.validate, &req) {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	session, err := h.auth.Register(ctx, req)
	if err != nil {
		respondError(c, err)
		return
	}
	h.writeSession(c, http.StatusCreated, session)
}

// Login handles POST /auth/login.
func (h *AuthController) Login(c *gin.Context) {
	var req models.UserLogin
	if !bindJSON(c, h.validate, &req) {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	session, err := h.auth.Login(ctx, req.Email, req.Password)
	if err != nil {
		respondError(c, err)
		return
	}
	h.writeSession(c, http.StatusOK, session)
}

// Refresh handles POST /auth/refresh: it trades the refresh token cookie for a new token pair.
func (h *AuthController) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie(refreshCookieName)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing refresh token"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	session, err := h.auth.Refresh(ctx, refreshToken)
	if err != nil {
		h.setRefreshCookie(c, "", -1)
		respondError(c, err)
		return
	}
	h.writeSession(c, http.StatusOK, session)
}

// Logout handles POST /auth/logout. The user is identified by the refresh token cookie, not the request body.
func (h *AuthController) Logout(c *gin.Context) {
	if refreshToken, err := c.Cookie(refreshCookieName); err == nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
		defer cancel()
		if err := h.auth.Logout(ctx, refreshToken); err != nil {
			respondError(c, err)
			return
		}
	}
	h.setRefreshCookie(c, "", -1)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// writeSession puts the refresh token in an HttpOnly cookie and the access token plus user data in the response body.
func (h *AuthController) writeSession(c *gin.Context, status int, s *services.Session) {
	h.setRefreshCookie(c, s.RefreshToken, int(tokens.RefreshTokenTTL.Seconds()))
	c.JSON(status, models.UserResponse{
		UserID:          s.User.ID.Hex(),
		FirstName:       s.User.FirstName,
		LastName:        s.User.LastName,
		Email:           s.User.Email,
		Role:            s.User.Role,
		AccessToken:     s.AccessToken,
		FavouriteGenres: s.User.FavouriteGenres,
	})
}

func (h *AuthController) setRefreshCookie(c *gin.Context, value string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshCookieName,
		Value:    value,
		Path:     refreshCookiePath,
		MaxAge:   maxAge,
		Secure:   h.cookie.Secure,
		HttpOnly: true,
		SameSite: h.cookie.SameSite,
	})
}
