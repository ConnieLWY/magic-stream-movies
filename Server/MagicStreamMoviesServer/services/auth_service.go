package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/models"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/repository"
	"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/tokens"
	"golang.org/x/crypto/bcrypt"
)

type UserStore interface {
	Create(ctx context.Context, user *models.User) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
	SetRefreshTokenHash(ctx context.Context, id, hash string) error
	RotateRefreshTokenHash(ctx context.Context, id, oldHash, newHash string) error
}

type GenreLookup interface {
	GenresByIDs(ctx context.Context, ids []int) ([]models.Genre, error)
}

// Session is the result of a successful login, registration or refresh.
type Session struct {
	User         *models.User
	AccessToken  string
	RefreshToken string
}

type AuthService struct {
	users  UserStore
	genres GenreLookup
	tokens *tokens.Manager
	// dummyHash lets Login run one bcrypt comparison even for unknown emails, so both failures take the same time and response timing cannot reveal which emails are registered.
	dummyHash []byte
}

func NewAuthService(users UserStore, genres GenreLookup, tm *tokens.Manager) *AuthService {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)
	return &AuthService{users: users, genres: genres, tokens: tm, dummyHash: dummy}
}

// NormalizeEmail trims surrounding spaces and lowercases the address. Both registration and login go through it.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// Register creates a regular user and signs them in (returns a token pair), so the frontend does not need a separate login call.
func (s *AuthService) Register(ctx context.Context, req models.UserRegister) (*Session, error) {
	// bcrypt only uses the first 72 bytes. validator's max=72 counts characters, and multi-byte characters can exceed 72 bytes, so check bytes here.
	if len(req.Password) > 72 {
		return nil, ErrPasswordTooLong
	}

	genres, err := s.genres.GenresByIDs(ctx, req.FavouriteGenreIDs)
	if err != nil {
		return nil, fmt.Errorf("look up genres: %w", err)
	}
	if len(genres) != len(req.FavouriteGenreIDs) {
		return nil, ErrUnknownGenre
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()
	user := &models.User{
		FirstName:       strings.TrimSpace(req.FirstName),
		LastName:        strings.TrimSpace(req.LastName),
		Email:           NormalizeEmail(req.Email),
		PasswordHash:    string(hash),
		Role:            models.RoleUser, // decided by the server, never trusted from the client
		CreatedAt:       now,
		UpdatedAt:       now,
		FavouriteGenres: genres,
	}

	// No "does this email exist?" query first: insert directly and let the unique index reject duplicates, avoiding a check-then-act race.
	if err := s.users.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return s.startSession(ctx, user)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*Session, error) {
	user, err := s.users.FindByEmail(ctx, NormalizeEmail(email))
	if errors.Is(err, repository.ErrNotFound) {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.startSession(ctx, user)
}

// Refresh trades a refresh token for a new token pair. Each refresh token can be used only once (rotation).
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*Session, error) {
	claims, err := s.tokens.ValidateRefresh(refreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// Read the latest role and name from the database instead of trusting the old token.
	user, err := s.users.FindByID(ctx, claims.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}

	accessToken, newRefreshToken, err := s.tokens.Generate(user.ID.Hex(), user.Role)
	if err != nil {
		return nil, err
	}

	err = s.users.RotateRefreshTokenHash(ctx, user.ID.Hex(), tokens.Hash(refreshToken), tokens.Hash(newRefreshToken))
	if errors.Is(err, repository.ErrNotFound) {
		// The stored hash has changed: the token was already used, the user logged out, or they signed in elsewhere.
		return nil, ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, fmt.Errorf("rotate refresh token: %w", err)
	}

	return &Session{User: user, AccessToken: accessToken, RefreshToken: newRefreshToken}, nil
}

// Logout revokes the refresh token on the server. An invalid token is a no-op (logging out always "succeeds").
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	claims, err := s.tokens.ValidateRefresh(refreshToken)
	if err != nil {
		return nil
	}
	err = s.users.SetRefreshTokenHash(ctx, claims.UserID, "")
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("clear refresh token: %w", err)
	}
	return nil
}

func (s *AuthService) startSession(ctx context.Context, user *models.User) (*Session, error) {
	accessToken, refreshToken, err := s.tokens.Generate(user.ID.Hex(), user.Role)
	if err != nil {
		return nil, err
	}
	if err := s.users.SetRefreshTokenHash(ctx, user.ID.Hex(), tokens.Hash(refreshToken)); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}
	return &Session{User: user, AccessToken: accessToken, RefreshToken: refreshToken}, nil
}
