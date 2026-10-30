package routes

import (
	"net/http"
	"time"

	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/controllers"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/middleware"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/models"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/tokens"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// Register sets up every route. Middleware is attached to route groups, so what is protected depends only on which group a route is in,
// not on call order, and unknown paths get 404 rather than 401.
func Register(router *gin.Engine, tm *tokens.Manager, auth *controllers.AuthController, movies *controllers.MovieController) {
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Login and register are rate limited per IP (10 per minute on average, bursts of 10) to slow down password guessing.
	// refresh and logout need a valid refresh token and are not limited (every page load calls refresh).
	authLimit := middleware.RateLimit(rate.Every(6*time.Second), 10)
	authGroup := router.Group("/auth")
	authGroup.POST("/register", authLimit, auth.Register)
	authGroup.POST("/login", authLimit, auth.Login)
	authGroup.POST("/refresh", auth.Refresh)
	authGroup.POST("/logout", auth.Logout)

	// Public
	router.GET("/movies", movies.ListMovies)
	router.GET("/genres", movies.ListGenres)

	// Signed-in users
	protected := router.Group("/", middleware.Auth(tm))
	protected.GET("/movies/:imdb_id", movies.GetMovie)
	protected.GET("/me/recommendations", movies.Recommended)

	// Admins only
	admin := protected.Group("/", middleware.RequireRole(models.RoleAdmin))
	admin.POST("/movies", movies.CreateMovie)
	admin.PATCH("/movies/:imdb_id/review", movies.UpdateReview)
}
