package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/middleware"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/models"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/services"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type MovieController struct {
	movies   *services.MovieService
	validate *validator.Validate
	timeout  time.Duration
	// reviewTimeout is longer than a normal request because writing a review waits for the LLM.
	reviewTimeout time.Duration
}

func NewMovieController(movies *services.MovieService, v *validator.Validate, timeout, reviewTimeout time.Duration) *MovieController {
	return &MovieController{movies: movies, validate: v, timeout: timeout, reviewTimeout: reviewTimeout}
}

// ListMovies handles GET /movies.
func (h *MovieController) ListMovies(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	movies, err := h.movies.List(ctx)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, movies)
}

// GetMovie handles GET /movies/:imdb_id.
func (h *MovieController) GetMovie(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	movie, err := h.movies.Get(ctx, c.Param("imdb_id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, movie)
}

// CreateMovie handles POST /movies (admins only).
func (h *MovieController) CreateMovie(c *gin.Context) {
	var req models.MovieCreate
	if !bindJSON(c, h.validate, &req) {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	movie, err := h.movies.Create(ctx, req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, movie)
}

// UpdateReview handles PATCH /movies/:imdb_id/review (admins only).
func (h *MovieController) UpdateReview(c *gin.Context) {
	var req models.ReviewUpdate
	if !bindJSON(c, h.validate, &req) {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.reviewTimeout)
	defer cancel()

	ranking, err := h.movies.UpdateReview(ctx, c.Param("imdb_id"), req.AdminReview)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, models.ReviewUpdateResponse{AdminReview: req.AdminReview, Ranking: ranking})
}

// Recommended handles GET /me/recommendations.
func (h *MovieController) Recommended(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	movies, err := h.movies.Recommended(ctx, middleware.UserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, movies)
}

// ListGenres handles GET /genres.
func (h *MovieController) ListGenres(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	genres, err := h.movies.Genres(ctx)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, genres)
}
