package repository_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/database"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/models"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/repository"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// These are integration tests against a real MongoDB:
//
//	MONGODB_TEST_URI=mongodb://localhost:27017 go test ./repository
//
// Each test uses a randomly named database that is dropped afterwards. Skipped when MONGODB_TEST_URI is not set.
func testDB(t *testing.T) *mongo.Database {
	t.Helper()
	uri := os.Getenv("MONGODB_TEST_URI")
	if uri == "" {
		t.Skip("MONGODB_TEST_URI not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := database.Connect(ctx, uri)
	if err != nil {
		t.Fatal(err)
	}
	db := client.Database(fmt.Sprintf("magic_stream_test_%d", time.Now().UnixNano()))
	if err := database.EnsureIndexes(ctx, db); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})
	return db
}

func TestUserRepository(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	users := repository.NewUserRepository(db)

	user := &models.User{Email: "ada@example.com", Role: models.RoleUser,
		FavouriteGenres: []models.Genre{{GenreID: 2, GenreName: "Drama"}}}
	if err := users.Create(ctx, user); err != nil {
		t.Fatal(err)
	}
	if err := users.Create(ctx, &models.User{Email: "ada@example.com"}); !errors.Is(err, repository.ErrDuplicate) {
		t.Errorf("duplicate email: err = %v", err)
	}

	id := user.ID.Hex()
	if _, err := users.FindByID(ctx, "not-an-object-id"); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("invalid id: err = %v", err)
	}

	if err := users.SetRefreshTokenHash(ctx, id, "h1"); err != nil {
		t.Fatal(err)
	}
	if err := users.RotateRefreshTokenHash(ctx, id, "h1", "h2"); err != nil {
		t.Fatalf("rotate with current hash: %v", err)
	}
	if err := users.RotateRefreshTokenHash(ctx, id, "h1", "h3"); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("rotate with stale hash: err = %v", err)
	}
	if err := users.SetRefreshTokenHash(ctx, id, ""); err != nil {
		t.Fatal(err)
	}
	found, _ := users.FindByID(ctx, id)
	if found.RefreshTokenHash != "" {
		t.Errorf("hash should be cleared, got %q", found.RefreshTokenHash)
	}

	names, err := users.FavouriteGenreNames(ctx, id)
	if err != nil || len(names) != 1 || names[0] != "Drama" {
		t.Errorf("names = %v, err = %v", names, err)
	}
}

func TestMovieRepository(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	movies := repository.NewMovieRepository(db)

	list, err := movies.List(ctx)
	if err != nil || list == nil || len(list) != 0 {
		t.Fatalf("empty collection should give an empty, non-nil slice: %#v, %v", list, err)
	}

	for i, rank := range []int{3, models.NotRankedValue, 1} {
		m := &models.Movie{ImdbID: fmt.Sprintf("tt000000%d", i), Title: "M",
			Genre: []models.Genre{{GenreID: 2, GenreName: "Drama"}}, Ranking: models.Ranking{RankingValue: rank}}
		if err := movies.Create(ctx, m); err != nil {
			t.Fatal(err)
		}
	}
	if err := movies.Create(ctx, &models.Movie{ImdbID: "tt0000000"}); !errors.Is(err, repository.ErrDuplicate) {
		t.Errorf("duplicate imdb_id: err = %v", err)
	}

	recs, err := movies.Recommend(ctx, []string{"Drama"}, 2)
	if err != nil || len(recs) != 2 || recs[0].Ranking.RankingValue != 1 || recs[1].Ranking.RankingValue != 3 {
		t.Errorf("recommend = %+v, err = %v", recs, err)
	}

	if err := movies.UpdateReview(ctx, "tt9999999", "x", models.Ranking{}); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("update missing movie: err = %v", err)
	}
}
