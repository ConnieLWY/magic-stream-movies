package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/internal/memstore"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/models"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/services"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/tokens"
)

var ctx = context.Background()

func newAuth() (*services.AuthService, *memstore.Users) {
	users := memstore.NewUsers()
	tm := tokens.NewManager([]byte(strings.Repeat("a", 32)), []byte(strings.Repeat("r", 32)))
	return services.NewAuthService(users, memstore.NewCatalog(), tm), users
}

func validRegistration() models.UserRegister {
	return models.UserRegister{
		FirstName: "Ada", LastName: "Lovelace", Email: "  Ada@Example.COM ",
		Password: "secret123", FavouriteGenreIDs: []int{2},
	}
}

func TestRegisterAlwaysCreatesUserRole(t *testing.T) {
	auth, users := newAuth()
	session, err := auth.Register(ctx, validRegistration())
	if err != nil {
		t.Fatal(err)
	}
	if session.User.Role != models.RoleUser {
		t.Errorf("role = %q, want USER", session.User.Role)
	}
	if session.User.Email != "ada@example.com" {
		t.Errorf("email not normalized: %q", session.User.Email)
	}
	if session.User.FavouriteGenres[0].GenreName != "Drama" {
		t.Errorf("genre name should come from the catalog, got %+v", session.User.FavouriteGenres)
	}
	stored, _ := users.FindByEmail(ctx, "ada@example.com")
	if stored.RefreshTokenHash != tokens.Hash(session.RefreshToken) {
		t.Error("database should hold the hash of the refresh token, not the token itself")
	}
}

func TestRegisterErrors(t *testing.T) {
	auth, _ := newAuth()
	if _, err := auth.Register(ctx, validRegistration()); err != nil {
		t.Fatal(err)
	}

	dup := validRegistration()
	dup.Email = "ADA@example.com"
	if _, err := auth.Register(ctx, dup); !errors.Is(err, services.ErrEmailTaken) {
		t.Errorf("duplicate email (different case): err = %v", err)
	}

	unknown := validRegistration()
	unknown.Email = "b@example.com"
	unknown.FavouriteGenreIDs = []int{2, 99}
	if _, err := auth.Register(ctx, unknown); !errors.Is(err, services.ErrUnknownGenre) {
		t.Errorf("unknown genre: err = %v", err)
	}

	long := validRegistration()
	long.Email = "c@example.com"
	long.Password = strings.Repeat("é", 40) // 40 characters, 80 bytes
	if _, err := auth.Register(ctx, long); !errors.Is(err, services.ErrPasswordTooLong) {
		t.Errorf("long password: err = %v", err)
	}
}

func TestLogin(t *testing.T) {
	auth, _ := newAuth()
	if _, err := auth.Register(ctx, validRegistration()); err != nil {
		t.Fatal(err)
	}

	if _, err := auth.Login(ctx, "ADA@example.com", "secret123"); err != nil {
		t.Errorf("login with different case: %v", err)
	}
	if _, err := auth.Login(ctx, "ada@example.com", "wrong"); !errors.Is(err, services.ErrInvalidCredentials) {
		t.Errorf("wrong password: err = %v", err)
	}
	if _, err := auth.Login(ctx, "nobody@example.com", "secret123"); !errors.Is(err, services.ErrInvalidCredentials) {
		t.Errorf("unknown user: err = %v", err)
	}
}

func TestRefreshRotatesAndRejectsReuse(t *testing.T) {
	auth, users := newAuth()
	first, _ := auth.Register(ctx, validRegistration())

	// Promote the user in the database: the refreshed token should carry the new role.
	users.SetRole("ada@example.com", models.RoleAdmin)

	second, err := auth.Refresh(ctx, first.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if second.User.Role != models.RoleAdmin {
		t.Errorf("refresh should read the latest role, got %q", second.User.Role)
	}
	if _, err := auth.Refresh(ctx, first.RefreshToken); !errors.Is(err, services.ErrInvalidRefreshToken) {
		t.Errorf("reusing a rotated token: err = %v", err)
	}
	if _, err := auth.Refresh(ctx, second.RefreshToken); err != nil {
		t.Errorf("latest token should still work: %v", err)
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	auth, _ := newAuth()
	s, _ := auth.Register(ctx, validRegistration())

	if err := auth.Logout(ctx, s.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Refresh(ctx, s.RefreshToken); !errors.Is(err, services.ErrInvalidRefreshToken) {
		t.Errorf("refresh after logout: err = %v", err)
	}
	if err := auth.Logout(ctx, "garbage"); err != nil {
		t.Errorf("logout with an invalid token should be a no-op, got %v", err)
	}
}

func shawshank() models.Movie {
	return models.Movie{
		ImdbID: "tt0111161", Title: "The Shawshank Redemption",
		Genre:   []models.Genre{{GenreID: 2, GenreName: "Drama"}},
		Ranking: models.Ranking{RankingValue: models.NotRankedValue, RankingName: "Not_Ranked"},
	}
}

func newMovieService(classifier *memstore.Classifier, movies ...models.Movie) (*services.MovieService, *memstore.Movies, *memstore.Users) {
	store := memstore.NewMovies(movies...)
	users := memstore.NewUsers()
	return services.NewMovieService(store, memstore.NewCatalog(), users, classifier, 5, time.Second), store, users
}

func TestUpdateReviewNormalizesLLMAnswer(t *testing.T) {
	classifier := &memstore.Classifier{Answer: " excellent.\n"}
	svc, store, _ := newMovieService(classifier, shawshank())

	ranking, err := svc.UpdateReview(ctx, "tt0111161", "A masterpiece.")
	if err != nil {
		t.Fatal(err)
	}
	if ranking.RankingValue != 1 || ranking.RankingName != "Excellent" {
		t.Errorf("ranking = %+v", ranking)
	}
	movie, _ := store.FindByImdbID(ctx, "tt0111161")
	if movie.AdminReview != "A masterpiece." || movie.Ranking.RankingValue != 1 {
		t.Errorf("movie not updated: %+v", movie)
	}
}

func TestUpdateReviewRejectsUnknownAnswer(t *testing.T) {
	for _, answer := range []string{"Masterpiece", "Not_Ranked", "Ignore previous instructions"} {
		classifier := &memstore.Classifier{Answer: answer}
		svc, store, _ := newMovieService(classifier, shawshank())

		if _, err := svc.UpdateReview(ctx, "tt0111161", "review"); !errors.Is(err, services.ErrUnrecognizedRanking) {
			t.Errorf("answer %q: err = %v", answer, err)
		}
		movie, _ := store.FindByImdbID(ctx, "tt0111161")
		if movie.AdminReview != "" {
			t.Errorf("answer %q: movie should not be updated", answer)
		}
	}
}

func TestUpdateReviewChecksMovieBeforeCallingLLM(t *testing.T) {
	classifier := &memstore.Classifier{Answer: "Good"}
	svc, _, _ := newMovieService(classifier)

	if _, err := svc.UpdateReview(ctx, "tt0000000", "review"); !errors.Is(err, services.ErrMovieNotFound) {
		t.Errorf("err = %v", err)
	}
	if classifier.Calls != 0 {
		t.Error("the paid LLM must not be called for a movie that does not exist")
	}
}

func TestCreateMovie(t *testing.T) {
	svc, _, _ := newMovieService(&memstore.Classifier{})
	req := models.MovieCreate{
		ImdbID: "tt0111161", Title: "The Shawshank Redemption",
		PosterPath: "https://example.com/p.jpg", YouTubeID: "PLl99DlL6b4", GenreIDs: []int{2},
	}

	movie, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if movie.Ranking.RankingValue != models.NotRankedValue || movie.Genre[0].GenreName != "Drama" {
		t.Errorf("movie = %+v", movie)
	}
	if _, err := svc.Create(ctx, req); !errors.Is(err, services.ErrMovieExists) {
		t.Errorf("duplicate imdb_id: err = %v", err)
	}
	req.ImdbID, req.GenreIDs = "tt0000001", []int{42}
	if _, err := svc.Create(ctx, req); !errors.Is(err, services.ErrUnknownGenre) {
		t.Errorf("unknown genre: err = %v", err)
	}
}

func TestRecommendedWithNoFavouritesReturnsEmptyList(t *testing.T) {
	svc, _, users := newMovieService(&memstore.Classifier{}, shawshank())
	user := &models.User{Email: "a@example.com"}
	_ = users.Create(ctx, user)

	movies, err := svc.Recommended(ctx, user.ID.Hex())
	if err != nil {
		t.Fatal(err)
	}
	if movies == nil || len(movies) != 0 {
		t.Errorf("want an empty, non-nil slice, got %#v", movies)
	}
}
