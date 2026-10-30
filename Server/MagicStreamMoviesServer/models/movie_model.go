package models

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

// NotRankedValue is the ranking_value for "not ranked yet". Recommendations sort by ranking_value ascending, so it must be the largest value.
const NotRankedValue = 999

type Genre struct {
	GenreID   int    `bson:"genre_id" json:"genre_id" validate:"required,min=1"`
	GenreName string `bson:"genre_name" json:"genre_name" validate:"required,min=2,max=100"`
}

type Ranking struct {
	RankingValue int    `bson:"ranking_value" json:"ranking_value" validate:"required"`
	RankingName  string `bson:"ranking_name" json:"ranking_name" validate:"required"`
}

type Movie struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	ImdbID      string        `bson:"imdb_id" json:"imdb_id" validate:"required,imdb_id"`
	Title       string        `bson:"title" json:"title" validate:"required,min=2,max=500"`
	PosterPath  string        `bson:"poster_path" json:"poster_path" validate:"required,url,max=2048"`
	YouTubeID   string        `bson:"youtube_id" json:"youtube_id" validate:"required,youtube_id"`
	Genre       []Genre       `bson:"genre" json:"genre" validate:"required,min=1,dive"`
	AdminReview string        `bson:"admin_review" json:"admin_review"`
	Ranking     Ranking       `bson:"ranking" json:"ranking"`
}

// MovieCreate is the request DTO for adding a movie. Review and ranking come from the review flow; new movies always start as not ranked.
type MovieCreate struct {
	ImdbID     string `json:"imdb_id" validate:"required,imdb_id"`
	Title      string `json:"title" validate:"required,min=2,max=500"`
	PosterPath string `json:"poster_path" validate:"required,url,max=2048"`
	YouTubeID  string `json:"youtube_id" validate:"required,youtube_id"`
	GenreIDs   []int  `json:"genre_ids" validate:"required,min=1,max=20,unique,dive,min=1"`
}

type ReviewUpdate struct {
	AdminReview string `json:"admin_review" validate:"required,min=1,max=2000"`
}

type ReviewUpdateResponse struct {
	AdminReview string  `json:"admin_review"`
	Ranking     Ranking `json:"ranking"`
}
