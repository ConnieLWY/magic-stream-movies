package routes_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/controllers"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/internal/memstore"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/models"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/routes"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/services"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/tokens"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/validation"
	"github.com/gin-gonic/gin"
)

type app struct {
	router *gin.Engine
	users  *memstore.Users
}

// newApp wires the real routes, middleware, controllers and services, swapping only the database and the LLM for in-memory fakes.
func newApp(t *testing.T) *app {
	t.Helper()
	gin.SetMode(gin.TestMode)

	users := memstore.NewUsers()
	catalog := memstore.NewCatalog()
	movies := memstore.NewMovies(models.Movie{
		ImdbID: "tt0111161", Title: "The Shawshank Redemption",
		Genre:   []models.Genre{{GenreID: 2, GenreName: "Drama"}},
		Ranking: models.Ranking{RankingValue: 1, RankingName: "Excellent"},
	})
	tm := tokens.NewManager([]byte(strings.Repeat("a", 32)), []byte(strings.Repeat("r", 32)))
	v := validation.New()

	authCtrl := controllers.NewAuthController(services.NewAuthService(users, catalog, tm), v,
		controllers.CookieOptions{Secure: true, SameSite: http.SameSiteLaxMode}, time.Second)
	movieCtrl := controllers.NewMovieController(
		services.NewMovieService(movies, catalog, users, &memstore.Classifier{Answer: "Good"}, 5, time.Second),
		v, time.Second, time.Second)

	router := gin.New()
	routes.Register(router, tm, authCtrl, movieCtrl)
	return &app{router: router, users: users}
}

func (a *app) do(method, path, body, accessToken string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	rec := httptest.NewRecorder()
	a.router.ServeHTTP(rec, req)
	return rec
}

// register signs up a user and returns their access token.
func (a *app) register(t *testing.T, email string) string {
	t.Helper()
	body := `{"first_name":"Ada","last_name":"Lovelace","email":"` + email + `","password":"secret123","favourite_genre_ids":[2],"role":"ADMIN"}`
	rec := a.do(http.MethodPost, "/auth/register", body, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", rec.Code, rec.Body)
	}
	var resp models.UserResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	return resp.AccessToken
}

func TestRegisterIgnoresClientRoleAndSetsScopedCookie(t *testing.T) {
	a := newApp(t)
	body := `{"first_name":"Eve","last_name":"Tester","email":"eve@example.com","password":"secret123","favourite_genre_ids":[2],"role":"ADMIN"}`
	rec := a.do(http.MethodPost, "/auth/register", body, "")

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	var resp models.UserResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Role != models.RoleUser || resp.AccessToken == "" {
		t.Errorf("response = %+v", resp)
	}

	cookie := rec.Result().Cookies()[0]
	if cookie.Name != "refresh_token" || !cookie.HttpOnly || !cookie.Secure || cookie.Path != "/auth" {
		t.Errorf("cookie = %+v", cookie)
	}
	if strings.Contains(rec.Body.String(), cookie.Value) {
		t.Error("refresh token must not appear in the response body")
	}
}

func TestValidationErrorsUseJSONFieldNames(t *testing.T) {
	a := newApp(t)
	rec := a.do(http.MethodPost, "/auth/login", `{"email":"not-an-email"}`, "")
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "password: required") {
		t.Errorf("status = %d, body = %s", rec.Code, rec.Body)
	}
}

func TestAuthorization(t *testing.T) {
	a := newApp(t)
	userToken := a.register(t, "user@example.com")
	movie := `{"imdb_id":"tt0068646","title":"The Godfather","poster_path":"https://example.com/p.jpg","youtube_id":"UaVTIH8mujA","genre_ids":[2]}`

	cases := []struct {
		name, method, path, body, token string
		want                            int
	}{
		{"public list", http.MethodGet, "/movies", "", "", http.StatusOK},
		{"public genres", http.MethodGet, "/genres", "", "", http.StatusOK},
		{"detail needs login", http.MethodGet, "/movies/tt0111161", "", "", http.StatusUnauthorized},
		{"detail as user", http.MethodGet, "/movies/tt0111161", "", userToken, http.StatusOK},
		{"missing movie", http.MethodGet, "/movies/tt9999999", "", userToken, http.StatusNotFound},
		{"bad token", http.MethodGet, "/movies/tt0111161", "", "not-a-jwt", http.StatusUnauthorized},
		{"recommendations", http.MethodGet, "/me/recommendations", "", userToken, http.StatusOK},
		{"create as user", http.MethodPost, "/movies", movie, userToken, http.StatusForbidden},
		{"review as user", http.MethodPatch, "/movies/tt0111161/review", `{"admin_review":"x"}`, userToken, http.StatusForbidden},
		{"unknown path", http.MethodGet, "/does-not-exist", "", "", http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if rec := a.do(tc.method, tc.path, tc.body, tc.token); rec.Code != tc.want {
				t.Errorf("status = %d, want %d (body %s)", rec.Code, tc.want, rec.Body)
			}
		})
	}
}

func TestAdminEndpoints(t *testing.T) {
	a := newApp(t)
	a.register(t, "admin@example.com")
	a.users.SetRole("admin@example.com", models.RoleAdmin)

	// The role lives in the token, so after promotion the user must log in again to get an ADMIN token.
	rec := a.do(http.MethodPost, "/auth/login", `{"email":"admin@example.com","password":"secret123"}`, "")
	var resp models.UserResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	token := resp.AccessToken

	movie := `{"imdb_id":"tt0068646","title":"The Godfather","poster_path":"https://example.com/p.jpg","youtube_id":"UaVTIH8mujA","genre_ids":[2]}`
	if rec := a.do(http.MethodPost, "/movies", movie, token); rec.Code != http.StatusCreated {
		t.Errorf("create: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(http.MethodPost, "/movies", movie, token); rec.Code != http.StatusConflict {
		t.Errorf("duplicate create: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(http.MethodPost, "/movies", `{"imdb_id":"bad"}`, token); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid movie: %d %s", rec.Code, rec.Body)
	}

	rec = a.do(http.MethodPatch, "/movies/tt0111161/review", `{"admin_review":"Solid."}`, token)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ranking_value":2`) {
		t.Errorf("review: %d %s", rec.Code, rec.Body)
	}
	if rec := a.do(http.MethodPatch, "/movies/tt0111161/review", `{"admin_review":""}`, token); rec.Code != http.StatusBadRequest {
		t.Errorf("empty review: %d %s", rec.Code, rec.Body)
	}
}

func TestLoginIsRateLimited(t *testing.T) {
	a := newApp(t)
	var last int
	// Use an invalid body: it fails validation with 400 before the slow bcrypt step, so the result does not depend on machine speed.
	for range 11 {
		last = a.do(http.MethodPost, "/auth/login", `{}`, "").Code
	}
	if last != http.StatusTooManyRequests {
		t.Errorf("11th attempt: status = %d, want 429", last)
	}
}
