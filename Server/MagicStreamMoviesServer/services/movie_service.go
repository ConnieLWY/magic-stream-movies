package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/models"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/repository"
)

type MovieStore interface {
	List(ctx context.Context) ([]models.Movie, error)
	FindByImdbID(ctx context.Context, imdbID string) (*models.Movie, error)
	Create(ctx context.Context, movie *models.Movie) error
	UpdateReview(ctx context.Context, imdbID, review string, ranking models.Ranking) error
	Recommend(ctx context.Context, genreNames []string, limit int64) ([]models.Movie, error)
}

type CatalogStore interface {
	GenreLookup
	Genres(ctx context.Context) ([]models.Genre, error)
	Rankings(ctx context.Context) ([]models.Ranking, error)
}

type FavouriteGenreReader interface {
	FavouriteGenreNames(ctx context.Context, userID string) ([]string, error)
}

// ReviewClassifier assigns a review to one of the ranking names in labels. OpenAI implements it in production; tests use a fake.
type ReviewClassifier interface {
	Classify(ctx context.Context, review string, labels []string) (string, error)
}

type MovieService struct {
	movies          MovieStore
	catalog         CatalogStore
	users           FavouriteGenreReader
	classifier      ReviewClassifier
	recommendLimit  int64
	classifyTimeout time.Duration
}

func NewMovieService(movies MovieStore, catalog CatalogStore, users FavouriteGenreReader,
	classifier ReviewClassifier, recommendLimit int64, classifyTimeout time.Duration) *MovieService {
	return &MovieService{
		movies:          movies,
		catalog:         catalog,
		users:           users,
		classifier:      classifier,
		recommendLimit:  recommendLimit,
		classifyTimeout: classifyTimeout,
	}
}

func (s *MovieService) List(ctx context.Context) ([]models.Movie, error) {
	return s.movies.List(ctx)
}

func (s *MovieService) Genres(ctx context.Context) ([]models.Genre, error) {
	return s.catalog.Genres(ctx)
}

func (s *MovieService) Get(ctx context.Context, imdbID string) (*models.Movie, error) {
	movie, err := s.movies.FindByImdbID(ctx, imdbID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrMovieNotFound
	}
	return movie, err
}

// Create adds a movie. Genre names are looked up in the genres collection and the ranking starts as "not ranked".
func (s *MovieService) Create(ctx context.Context, req models.MovieCreate) (*models.Movie, error) {
	genres, err := s.catalog.GenresByIDs(ctx, req.GenreIDs)
	if err != nil {
		return nil, fmt.Errorf("look up genres: %w", err)
	}
	if len(genres) != len(req.GenreIDs) {
		return nil, ErrUnknownGenre
	}

	notRanked, err := s.rankingByValue(ctx, models.NotRankedValue)
	if err != nil {
		return nil, err
	}

	movie := &models.Movie{
		ImdbID:     req.ImdbID,
		Title:      strings.TrimSpace(req.Title),
		PosterPath: req.PosterPath,
		YouTubeID:  req.YouTubeID,
		Genre:      genres,
		Ranking:    notRanked,
	}
	if err := s.movies.Create(ctx, movie); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, ErrMovieExists
		}
		return nil, fmt.Errorf("create movie: %w", err)
	}
	return movie, nil
}

// UpdateReview saves an admin review and lets the LLM pick one of the existing rankings.
func (s *MovieService) UpdateReview(ctx context.Context, imdbID, review string) (models.Ranking, error) {
	// Make sure the movie exists before calling the pay-per-use LLM.
	if _, err := s.Get(ctx, imdbID); err != nil {
		return models.Ranking{}, err
	}

	rankings, err := s.catalog.Rankings(ctx)
	if err != nil {
		return models.Ranking{}, fmt.Errorf("load rankings: %w", err)
	}
	labels := make([]string, 0, len(rankings))
	for _, r := range rankings {
		if r.RankingValue != models.NotRankedValue {
			labels = append(labels, r.RankingName)
		}
	}

	classifyCtx, cancel := context.WithTimeout(ctx, s.classifyTimeout)
	defer cancel()
	answer, err := s.classifier.Classify(classifyCtx, review, labels)
	if err != nil {
		return models.Ranking{}, fmt.Errorf("classify review: %w", err)
	}

	// LLM output is untrusted: only accept an answer that matches an existing ranking name, otherwise refuse to save.
	ranking, ok := matchRanking(answer, rankings)
	if !ok {
		return models.Ranking{}, ErrUnrecognizedRanking
	}

	err = s.movies.UpdateReview(ctx, imdbID, review, ranking)
	if errors.Is(err, repository.ErrNotFound) {
		return models.Ranking{}, ErrMovieNotFound
	}
	if err != nil {
		return models.Ranking{}, fmt.Errorf("update review: %w", err)
	}
	return ranking, nil
}

// Recommended returns movies in the user's favourite genres.
func (s *MovieService) Recommended(ctx context.Context, userID string) ([]models.Movie, error) {
	genreNames, err := s.users.FavouriteGenreNames(ctx, userID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("load favourite genres: %w", err)
	}
	if len(genreNames) == 0 {
		return []models.Movie{}, nil
	}
	return s.movies.Recommend(ctx, genreNames, s.recommendLimit)
}

func (s *MovieService) rankingByValue(ctx context.Context, value int) (models.Ranking, error) {
	rankings, err := s.catalog.Rankings(ctx)
	if err != nil {
		return models.Ranking{}, fmt.Errorf("load rankings: %w", err)
	}
	for _, r := range rankings {
		if r.RankingValue == value {
			return r, nil
		}
	}
	return models.Ranking{}, fmt.Errorf("ranking %d is missing from the rankings collection", value)
}

// matchRanking compares the LLM answer with ranking names after trimming whitespace, quotes and trailing punctuation, ignoring case.
// "Not ranked" is never a valid classification.
func matchRanking(answer string, rankings []models.Ranking) (models.Ranking, bool) {
	normalized := strings.Trim(strings.TrimSpace(answer), "\"'`*.!。！ \n")
	for _, r := range rankings {
		if r.RankingValue != models.NotRankedValue && strings.EqualFold(normalized, r.RankingName) {
			return r, true
		}
	}
	return models.Ranking{}, false
}
