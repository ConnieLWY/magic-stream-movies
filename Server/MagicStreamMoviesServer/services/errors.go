// Package services implements the business logic. It depends on the small interfaces defined below rather than concrete database types,
// so tests can swap in in-memory fakes for MongoDB and OpenAI.
package services

import "errors"

// These errors mean "the request itself is wrong"; controllers map them to 4xx status codes.
// Every other error is treated as an internal server error.
var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrEmailTaken          = errors.New("email is already registered")
	ErrPasswordTooLong     = errors.New("password must be at most 72 bytes")
	ErrUnknownGenre        = errors.New("one or more genres do not exist")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
	ErrMovieNotFound       = errors.New("movie not found")
	ErrMovieExists         = errors.New("a movie with this imdb_id already exists")
	ErrUnrecognizedRanking = errors.New("review classifier returned an unknown ranking")
)
