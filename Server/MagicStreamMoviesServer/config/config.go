// Package config reads and validates every environment variable once, at startup.
// Other packages receive a *Config and never call os.Getenv or godotenv.Load themselves.
package config

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// MinSecretLength is the minimum JWT signing key length in bytes. An HS256 key should be at least as long as the hash output (32 bytes).
const MinSecretLength = 32

type Config struct {
	Port             string
	MongoURI         string
	DatabaseName     string
	SecretKey        []byte
	SecretRefreshKey []byte
	AllowedOrigins   []string
	TrustedProxies   []string

	OpenAIAPIKey       string
	OpenAIModel        string
	BasePromptTemplate string
	OpenAITimeout      time.Duration

	RecommendedMovieLimit int64
	RequestTimeout        time.Duration

	CookieSecure   bool
	CookieSameSite http.SameSite
}

// Load builds a Config from the environment. It returns an error for missing or malformed settings and leaves exiting to main.
// Every problem is reported at once rather than stopping at the first.
func Load() (*Config, error) {
	var errs []error

	required := func(key string) string {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			errs = append(errs, fmt.Errorf("%s is required", key))
		}
		return v
	}

	cfg := &Config{
		Port:               envOr("PORT", "8080"),
		MongoURI:           required("MONGODB_URI"),
		DatabaseName:       required("DATABASE_NAME"),
		OpenAIAPIKey:       required("OPENAI_API_KEY"),
		OpenAIModel:        envOr("OPENAI_MODEL", "gpt-4.1-mini"),
		BasePromptTemplate: required("BASE_PROMPT_TEMPLATE"),
		AllowedOrigins:     splitList(envOr("ALLOWED_ORIGINS", "http://localhost:5173")),
		TrustedProxies:     splitList(os.Getenv("TRUSTED_PROXIES")),
	}

	cfg.SecretKey = []byte(required("SECRET_KEY"))
	cfg.SecretRefreshKey = []byte(required("SECRET_REFRESH_KEY"))
	if n := len(cfg.SecretKey); n > 0 && n < MinSecretLength {
		errs = append(errs, fmt.Errorf("SECRET_KEY must be at least %d bytes, got %d", MinSecretLength, n))
	}
	if n := len(cfg.SecretRefreshKey); n > 0 && n < MinSecretLength {
		errs = append(errs, fmt.Errorf("SECRET_REFRESH_KEY must be at least %d bytes, got %d", MinSecretLength, n))
	}
	if len(cfg.SecretKey) > 0 && string(cfg.SecretKey) == string(cfg.SecretRefreshKey) {
		errs = append(errs, errors.New("SECRET_KEY and SECRET_REFRESH_KEY must be different"))
	}

	if cfg.BasePromptTemplate != "" && !strings.Contains(cfg.BasePromptTemplate, "{rankings}") {
		errs = append(errs, errors.New("BASE_PROMPT_TEMPLATE must contain the {rankings} placeholder"))
	}

	var err error
	if cfg.RecommendedMovieLimit, err = positiveInt("RECOMMENDED_MOVIE_LIMIT", 5); err != nil {
		errs = append(errs, err)
	}
	if cfg.RequestTimeout, err = duration("REQUEST_TIMEOUT", 10*time.Second); err != nil {
		errs = append(errs, err)
	}
	if cfg.OpenAITimeout, err = duration("OPENAI_TIMEOUT", 30*time.Second); err != nil {
		errs = append(errs, err)
	}
	if cfg.CookieSecure, err = boolean("COOKIE_SECURE", true); err != nil {
		errs = append(errs, err)
	}
	if cfg.CookieSameSite, err = sameSite(envOr("COOKIE_SAMESITE", "Lax")); err != nil {
		errs = append(errs, err)
	}
	if cfg.CookieSameSite == http.SameSiteNoneMode && !cfg.CookieSecure {
		errs = append(errs, errors.New("COOKIE_SAMESITE=None requires COOKIE_SECURE=true"))
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func positiveInt(key string, fallback int64) (int64, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer, got %q", key, v)
	}
	return n, nil
}

func duration(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration such as 10s, got %q", key, v)
	}
	return d, nil
}

func boolean(key string, fallback bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false, got %q", key, v)
	}
	return b, nil
}

func sameSite(v string) (http.SameSite, error) {
	switch strings.ToLower(v) {
	case "lax":
		return http.SameSiteLaxMode, nil
	case "strict":
		return http.SameSiteStrictMode, nil
	case "none":
		return http.SameSiteNoneMode, nil
	}
	return 0, fmt.Errorf("COOKIE_SAMESITE must be Lax, Strict or None, got %q", v)
}
