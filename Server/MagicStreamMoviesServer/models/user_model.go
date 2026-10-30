package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	RoleAdmin = "ADMIN"
	RoleUser  = "USER"
)

// User is a document in the users collection. It stays inside the server: it is never bound from a request body or returned as a response.
// The user ID is MongoDB's _id (exposed as a hex string).
type User struct {
	ID               bson.ObjectID `bson:"_id,omitempty" json:"-"`
	FirstName        string        `bson:"first_name" json:"-"`
	LastName         string        `bson:"last_name" json:"-"`
	Email            string        `bson:"email" json:"-"`
	PasswordHash     string        `bson:"password" json:"-"`
	Role             string        `bson:"role" json:"-"`
	CreatedAt        time.Time     `bson:"created_at" json:"-"`
	UpdatedAt        time.Time     `bson:"updated_at" json:"-"`
	RefreshTokenHash string        `bson:"refresh_token_hash,omitempty" json:"-"`
	FavouriteGenres  []Genre       `bson:"favourite_genres" json:"-"`
}

// UserRegister is the registration request DTO. It deliberately has no Role or other server-controlled fields; if a client sends them they are ignored.
// Favourite genres are submitted as a list of genre_id values; the server looks up their names in the genres collection.
type UserRegister struct {
	FirstName         string `json:"first_name" validate:"required,min=2,max=100"`
	LastName          string `json:"last_name" validate:"required,min=2,max=100"`
	Email             string `json:"email" validate:"required,email,max=254"`
	Password          string `json:"password" validate:"required,min=8,max=72"`
	FavouriteGenreIDs []int  `json:"favourite_genre_ids" validate:"required,min=1,max=20,unique,dive,min=1"`
}

type UserLogin struct {
	Email    string `json:"email" validate:"required,email,max=254"`
	Password string `json:"password" validate:"required,max=72"`
}

// UserResponse is the user data sent to the frontend: no password hash and no refresh token.
type UserResponse struct {
	UserID          string  `json:"user_id"`
	FirstName       string  `json:"first_name"`
	LastName        string  `json:"last_name"`
	Email           string  `json:"email"`
	Role            string  `json:"role"`
	AccessToken     string  `json:"access_token"`
	FavouriteGenres []Genre `json:"favourite_genres"`
}
