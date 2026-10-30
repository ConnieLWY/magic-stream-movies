package repository

import (
	"context"

	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// CatalogRepository reads the two read-only lookup collections: genres and rankings.
type CatalogRepository struct {
	genres   *mongo.Collection
	rankings *mongo.Collection
}

func NewCatalogRepository(db *mongo.Database) *CatalogRepository {
	return &CatalogRepository{genres: db.Collection("genres"), rankings: db.Collection("rankings")}
}

func (r *CatalogRepository) Genres(ctx context.Context) ([]models.Genre, error) {
	return findAll[models.Genre](ctx, r.genres, bson.M{},
		options.Find().SetSort(bson.D{{Key: "genre_id", Value: 1}}))
}

// GenresByIDs returns the genres whose IDs are in the list. Unknown IDs are skipped; callers compare counts to detect them.
func (r *CatalogRepository) GenresByIDs(ctx context.Context, ids []int) ([]models.Genre, error) {
	return findAll[models.Genre](ctx, r.genres, bson.M{"genre_id": bson.M{"$in": ids}},
		options.Find().SetSort(bson.D{{Key: "genre_id", Value: 1}}))
}

func (r *CatalogRepository) Rankings(ctx context.Context) ([]models.Ranking, error) {
	return findAll[models.Ranking](ctx, r.rankings, bson.M{},
		options.Find().SetSort(bson.D{{Key: "ranking_value", Value: 1}}))
}
