// seed prepares the database: it creates indexes and loads genres, rankings and sample movies.
//
//	go run ./cmd/seed                         # load seed data (existing records are overwritten with the seed version)
//	go run ./cmd/seed -admin you@example.com  # also promote this registered user to ADMIN
//
// Safe to run repeatedly: every record is upserted by its unique key, so nothing is duplicated.
package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/database"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/models"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

//go:embed data/*.json
var data embed.FS

func main() {
	admin := flag.String("admin", "", "email of an existing user to promote to ADMIN")
	flag.Parse()

	if err := run(*admin); err != nil {
		log.Fatal(err)
	}
}

func run(adminEmail string) error {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	uri, dbName := os.Getenv("MONGODB_URI"), os.Getenv("DATABASE_NAME")
	if uri == "" || dbName == "" {
		return errors.New("MONGODB_URI and DATABASE_NAME are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	client, err := database.Connect(ctx, uri)
	if err != nil {
		return err
	}
	defer client.Disconnect(context.Background())
	db := client.Database(dbName)

	if err := database.EnsureIndexes(ctx, db); err != nil {
		return err
	}

	if err := upsertAll[models.Genre](ctx, db, "genres", "genre_id", func(g models.Genre) any { return g.GenreID }); err != nil {
		return err
	}
	if err := upsertAll[models.Ranking](ctx, db, "rankings", "ranking_value", func(r models.Ranking) any { return r.RankingValue }); err != nil {
		return err
	}
	if err := upsertAll[models.Movie](ctx, db, "movies", "imdb_id", func(m models.Movie) any { return m.ImdbID }); err != nil {
		return err
	}

	// Remove user fields left over from older versions: plaintext tokens, the misspelled update_at, and user_id (duplicates _id).
	result, err := db.Collection("users").UpdateMany(ctx, bson.M{}, bson.M{"$unset": bson.M{
		"token": "", "refresh_token": "", "update_at": "", "user_id": "",
	}})
	if err != nil {
		return err
	}
	log.Printf("users: cleaned legacy fields on %d documents", result.ModifiedCount)

	if adminEmail != "" {
		email := strings.ToLower(strings.TrimSpace(adminEmail))
		result, err := db.Collection("users").UpdateOne(ctx, bson.M{"email": email},
			bson.M{"$set": bson.M{"role": models.RoleAdmin, "updated_at": time.Now()}})
		if err != nil {
			return err
		}
		if result.MatchedCount == 0 {
			return fmt.Errorf("no user with email %s; register first, then run seed again", email)
		}
		log.Printf("%s is now an ADMIN (log in again to get a token with the new role)", email)
	}
	return nil
}

// upsertAll reads data/<collection>.json and upserts each document by the key field.
func upsertAll[T any](ctx context.Context, db *mongo.Database, collection, key string, keyOf func(T) any) error {
	raw, err := data.ReadFile("data/" + collection + ".json")
	if err != nil {
		return err
	}
	var docs []T
	if err := json.Unmarshal(raw, &docs); err != nil {
		return fmt.Errorf("parse %s.json: %w", collection, err)
	}

	coll := db.Collection(collection)
	for _, doc := range docs {
		_, err := coll.ReplaceOne(ctx, bson.M{key: keyOf(doc)}, doc, options.Replace().SetUpsert(true))
		if err != nil {
			return fmt.Errorf("upsert into %s: %w", collection, err)
		}
	}
	log.Printf("%s: %d documents", collection, len(docs))
	return nil
}
