package config

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func setValidEnv(t *testing.T) {
	t.Helper()
	for k, v := range map[string]string{
		"MONGODB_URI":             "mongodb://localhost:27017",
		"DATABASE_NAME":           "test",
		"SECRET_KEY":              strings.Repeat("a", 32),
		"SECRET_REFRESH_KEY":      strings.Repeat("b", 32),
		"OPENAI_API_KEY":          "sk-test",
		"BASE_PROMPT_TEMPLATE":    "Pick one of {rankings}: ",
		"RECOMMENDED_MOVIE_LIMIT": "",
		"PORT":                    "",
		"COOKIE_SAMESITE":         "",
		"COOKIE_SECURE":           "",
	} {
		t.Setenv(k, v)
	}
}

func TestLoadDefaults(t *testing.T) {
	setValidEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "8080" || cfg.RecommendedMovieLimit != 5 || cfg.RequestTimeout != 10*time.Second {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
	if !cfg.CookieSecure || cfg.CookieSameSite != http.SameSiteLaxMode {
		t.Errorf("unexpected cookie defaults: secure=%v samesite=%v", cfg.CookieSecure, cfg.CookieSameSite)
	}
}

func TestLoadRejectsBadConfig(t *testing.T) {
	cases := map[string]map[string]string{
		"missing uri":         {"MONGODB_URI": ""},
		"short secret":        {"SECRET_KEY": "short"},
		"same secrets":        {"SECRET_REFRESH_KEY": strings.Repeat("a", 32)},
		"bad limit":           {"RECOMMENDED_MOVIE_LIMIT": "five"},
		"zero limit":          {"RECOMMENDED_MOVIE_LIMIT": "0"},
		"prompt w/o rankings": {"BASE_PROMPT_TEMPLATE": "no placeholder"},
		"bad samesite":        {"COOKIE_SAMESITE": "sometimes"},
		"none without secure": {"COOKIE_SAMESITE": "None", "COOKIE_SECURE": "false"},
	}
	for name, overrides := range cases {
		t.Run(name, func(t *testing.T) {
			setValidEnv(t)
			for k, v := range overrides {
				t.Setenv(k, v)
			}
			if _, err := Load(); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestLoadReportsAllProblems(t *testing.T) {
	setValidEnv(t)
	t.Setenv("MONGODB_URI", "")
	t.Setenv("DATABASE_NAME", "")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "MONGODB_URI") || !strings.Contains(err.Error(), "DATABASE_NAME") {
		t.Errorf("error should list every missing variable, got %v", err)
	}
}
