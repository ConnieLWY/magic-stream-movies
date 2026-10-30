// Package memstore provides in-memory implementations of the storage interfaces used by services, for tests only.
// It mimics MongoDB's unique indexes (returning repository.ErrDuplicate on conflicts).
package memstore

import (
	"context"
	"slices"
	"sync"

	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/models"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/repository"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Users struct {
	mu    sync.Mutex
	users map[string]*models.User // key: ID.Hex()
}

func NewUsers() *Users { return &Users{users: map[string]*models.User{}} }

func (s *Users) Create(_ context.Context, user *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.users {
		if u.Email == user.Email {
			return repository.ErrDuplicate
		}
	}
	user.ID = bson.NewObjectID()
	clone := *user
	s.users[user.ID.Hex()] = &clone
	return nil
}

func (s *Users) FindByEmail(_ context.Context, email string) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.users {
		if u.Email == email {
			clone := *u
			return &clone, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (s *Users) FindByID(_ context.Context, id string) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	clone := *u
	return &clone, nil
}

func (s *Users) SetRefreshTokenHash(_ context.Context, id, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[id]
	if !ok {
		return repository.ErrNotFound
	}
	u.RefreshTokenHash = hash
	return nil
}

func (s *Users) RotateRefreshTokenHash(_ context.Context, id, oldHash, newHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[id]
	if !ok || u.RefreshTokenHash != oldHash {
		return repository.ErrNotFound
	}
	u.RefreshTokenHash = newHash
	return nil
}

func (s *Users) FavouriteGenreNames(ctx context.Context, id string) ([]string, error) {
	u, err := s.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	names := []string{}
	for _, g := range u.FavouriteGenres {
		names = append(names, g.GenreName)
	}
	return names, nil
}

// SetRole changes a user's role directly, like promoting an admin by hand in the database.
func (s *Users) SetRole(email, role string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.users {
		if u.Email == email {
			u.Role = role
		}
	}
}

type Movies struct {
	mu     sync.Mutex
	movies []models.Movie
}

func NewMovies(movies ...models.Movie) *Movies { return &Movies{movies: movies} }

func (s *Movies) List(context.Context) ([]models.Movie, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.movies), nil
}

func (s *Movies) FindByImdbID(_ context.Context, imdbID string) (*models.Movie, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.movies {
		if m.ImdbID == imdbID {
			return &m, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (s *Movies) Create(_ context.Context, movie *models.Movie) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.movies {
		if m.ImdbID == movie.ImdbID {
			return repository.ErrDuplicate
		}
	}
	movie.ID = bson.NewObjectID()
	s.movies = append(s.movies, *movie)
	return nil
}

func (s *Movies) UpdateReview(_ context.Context, imdbID, review string, ranking models.Ranking) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.movies {
		if s.movies[i].ImdbID == imdbID {
			s.movies[i].AdminReview = review
			s.movies[i].Ranking = ranking
			return nil
		}
	}
	return repository.ErrNotFound
}

func (s *Movies) Recommend(_ context.Context, genreNames []string, limit int64) ([]models.Movie, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []models.Movie{}
	for _, m := range s.movies {
		if slices.ContainsFunc(m.Genre, func(g models.Genre) bool { return slices.Contains(genreNames, g.GenreName) }) {
			out = append(out, m)
		}
	}
	slices.SortStableFunc(out, func(a, b models.Movie) int { return a.Ranking.RankingValue - b.Ranking.RankingValue })
	if int64(len(out)) > limit {
		out = out[:limit]
	}
	return out, nil
}

type Catalog struct {
	GenreList   []models.Genre
	RankingList []models.Ranking
}

// NewCatalog returns the same genres and rankings as the seed data.
func NewCatalog() *Catalog {
	return &Catalog{
		GenreList: []models.Genre{
			{GenreID: 1, GenreName: "Comedy"}, {GenreID: 2, GenreName: "Drama"}, {GenreID: 3, GenreName: "Western"},
		},
		RankingList: []models.Ranking{
			{RankingValue: 1, RankingName: "Excellent"}, {RankingValue: 2, RankingName: "Good"},
			{RankingValue: 3, RankingName: "Okay"}, {RankingValue: 4, RankingName: "Bad"},
			{RankingValue: 5, RankingName: "Terrible"}, {RankingValue: models.NotRankedValue, RankingName: "Not_Ranked"},
		},
	}
}

func (c *Catalog) Genres(context.Context) ([]models.Genre, error) {
	return slices.Clone(c.GenreList), nil
}

func (c *Catalog) GenresByIDs(_ context.Context, ids []int) ([]models.Genre, error) {
	out := []models.Genre{}
	for _, g := range c.GenreList {
		if slices.Contains(ids, g.GenreID) {
			out = append(out, g)
		}
	}
	return out, nil
}

func (c *Catalog) Rankings(context.Context) ([]models.Ranking, error) {
	return slices.Clone(c.RankingList), nil
}

// Classifier is a scriptable ReviewClassifier: it returns a preset answer and counts its calls.
type Classifier struct {
	mu     sync.Mutex
	Answer string
	Err    error
	Calls  int
}

func (c *Classifier) Classify(ctx context.Context, _ string, _ []string) (string, error) {
	c.mu.Lock()
	c.Calls++
	answer, err := c.Answer, c.Err
	c.mu.Unlock()
	if err != nil {
		return "", err
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	return answer, nil
}
