package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/config"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/controllers"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/database"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/middleware"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/repository"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/routes"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/services"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/tokens"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/validation"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Request body limit. The largest request is a review (2000 characters), so 1 MB is plenty.
const maxBodyBytes = 1 << 20

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run holds all startup logic. It returns an error instead of calling log.Fatal so that deferred cleanup (e.g. disconnecting from MongoDB) still runs.
func run() error {
	// Load .env exactly once, here. A missing file is fine (e.g. Docker injects the variables directly).
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	startupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := database.Connect(startupCtx, cfg.MongoURI)
	if err != nil {
		return err
	}
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Printf("disconnect mongodb: %v", err)
		}
	}()

	db := client.Database(cfg.DatabaseName)
	if err := database.EnsureIndexes(startupCtx, db); err != nil {
		return err
	}

	classifier, err := services.NewOpenAIClassifier(cfg.OpenAIAPIKey, cfg.OpenAIModel, cfg.BasePromptTemplate)
	if err != nil {
		return err
	}

	userRepo := repository.NewUserRepository(db)
	catalogRepo := repository.NewCatalogRepository(db)
	tokenManager := tokens.NewManager(cfg.SecretKey, cfg.SecretRefreshKey)
	validate := validation.New()

	authService := services.NewAuthService(userRepo, catalogRepo, tokenManager)
	movieService := services.NewMovieService(repository.NewMovieRepository(db), catalogRepo, userRepo,
		classifier, cfg.RecommendedMovieLimit, cfg.OpenAITimeout)

	authController := controllers.NewAuthController(authService, validate,
		controllers.CookieOptions{Secure: cfg.CookieSecure, SameSite: cfg.CookieSameSite}, cfg.RequestTimeout)
	movieController := controllers.NewMovieController(movieService, validate,
		cfg.RequestTimeout, cfg.RequestTimeout+cfg.OpenAITimeout)

	router := gin.New()
	// Trust no proxy's X-Forwarded-For, otherwise clients could spoof their IP to dodge rate limiting. List your proxies here when deploying behind one.
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return err
	}
	router.Use(
		gin.Logger(),
		gin.Recovery(),
		cors.New(cors.Config{
			AllowOrigins:     cfg.AllowedOrigins,
			AllowMethods:     []string{"GET", "POST", "PATCH", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}),
		middleware.SecurityHeaders(),
		middleware.BodyLimit(maxBodyBytes),
	)
	routes.Register(router, tokenManager, authController, movieController)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      cfg.RequestTimeout + cfg.OpenAITimeout + 5*time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// On Ctrl+C or SIGTERM (docker stop), shut down gracefully: stop accepting connections and let in-flight requests finish.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", server.Addr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		log.Println("shutting down...")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShutdown()
	return server.Shutdown(shutdownCtx)
}
