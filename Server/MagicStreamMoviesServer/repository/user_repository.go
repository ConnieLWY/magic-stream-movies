package repository

import (
	"context"
	"time"

	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type UserRepository struct {
	coll *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{coll: db.Collection("users")}
}

// Create inserts a new user and fills in user.ID. A duplicate email returns ErrDuplicate (via the unique index on email).
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	user.ID = bson.NewObjectID()
	_, err := r.coll.InsertOne(ctx, user)
	return translate(err)
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.coll.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, translate(err)
	}
	return &user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrNotFound
	}
	var user models.User
	if err := r.coll.FindOne(ctx, bson.M{"_id": oid}).Decode(&user); err != nil {
		return nil, translate(err)
	}
	return &user, nil
}

// SetRefreshTokenHash stores the hash of the current refresh token, or clears it when hash is empty.
func (r *UserRepository) SetRefreshTokenHash(ctx context.Context, id, hash string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return ErrNotFound
	}
	update := bson.M{"$set": bson.M{"updated_at": time.Now()}}
	if hash == "" {
		update["$unset"] = bson.M{"refresh_token_hash": ""}
	} else {
		update["$set"].(bson.M)["refresh_token_hash"] = hash
	}
	result, err := r.coll.UpdateOne(ctx, bson.M{"_id": oid}, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// RotateRefreshTokenHash replaces the hash with newHash only if it still equals oldHash.
// The check and the update are one statement, so of two concurrent requests holding the same old token only one succeeds.
func (r *UserRepository) RotateRefreshTokenHash(ctx context.Context, id, oldHash, newHash string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return ErrNotFound
	}
	result, err := r.coll.UpdateOne(ctx,
		bson.M{"_id": oid, "refresh_token_hash": oldHash},
		bson.M{"$set": bson.M{"refresh_token_hash": newHash, "updated_at": time.Now()}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// FavouriteGenreNames returns just the names of the user's favourite genres. Returns ErrNotFound for an unknown user.
func (r *UserRepository) FavouriteGenreNames(ctx context.Context, id string) ([]string, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrNotFound
	}
	var doc struct {
		FavouriteGenres []models.Genre `bson:"favourite_genres"`
	}
	opts := options.FindOne().SetProjection(bson.M{"favourite_genres.genre_name": 1})
	if err := r.coll.FindOne(ctx, bson.M{"_id": oid}, opts).Decode(&doc); err != nil {
		return nil, translate(err)
	}
	names := make([]string, 0, len(doc.FavouriteGenres))
	for _, g := range doc.FavouriteGenres {
		names = append(names, g.GenreName)
	}
	return names, nil
}
