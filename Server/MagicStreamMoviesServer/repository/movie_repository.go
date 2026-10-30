package repository

import (
	"context"

	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MovieRepository struct {
	coll *mongo.Collection
}

func NewMovieRepository(db *mongo.Database) *MovieRepository {
	return &MovieRepository{coll: db.Collection("movies")}
}

func (r *MovieRepository) List(ctx context.Context) ([]models.Movie, error) {
	return findAll[models.Movie](ctx, r.coll, bson.M{})
}

func (r *MovieRepository) FindByImdbID(ctx context.Context, imdbID string) (*models.Movie, error) {
	var movie models.Movie
	if err := r.coll.FindOne(ctx, bson.M{"imdb_id": imdbID}).Decode(&movie); err != nil {
		return nil, translate(err)
	}
	return &movie, nil
}

// Create inserts a movie. A duplicate imdb_id returns ErrDuplicate (via the unique index on imdb_id).
func (r *MovieRepository) Create(ctx context.Context, movie *models.Movie) error {
	movie.ID = bson.NewObjectID()
	_, err := r.coll.InsertOne(ctx, movie)
	return translate(err)
}

// UpdateReview sets the review and ranking. Returns ErrNotFound if the movie does not exist.
func (r *MovieRepository) UpdateReview(ctx context.Context, imdbID, review string, ranking models.Ranking) error {
	result, err := r.coll.UpdateOne(ctx,
		bson.M{"imdb_id": imdbID},
		bson.M{"$set": bson.M{"admin_review": review, "ranking": ranking}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// Recommend returns movies in any of the given genres, best ranking first.
func (r *MovieRepository) Recommend(ctx context.Context, genreNames []string, limit int64) ([]models.Movie, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "ranking.ranking_value", Value: 1}}).
		SetLimit(limit)
	return findAll[models.Movie](ctx, r.coll, bson.M{"genre.genre_name": bson.M{"$in": genreNames}}, opts)
}

// findAll runs a query and decodes every result. With no results it returns an empty slice (JSON [] rather than null).
func findAll[T any](ctx context.Context, coll *mongo.Collection, filter any, opts ...options.Lister[options.FindOptions]) ([]T, error) {
	cursor, err := coll.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	items := []T{}
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}
