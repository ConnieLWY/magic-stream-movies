// Package repository wraps all MongoDB access. The layer above (services) only sees domain models and the errors defined here,
// never bson, cursors or other driver details.
package repository

import (
	"errors"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrDuplicate = errors.New("duplicate key")
)

// translate maps driver errors to this package's errors and returns anything else unchanged.
func translate(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, mongo.ErrNoDocuments):
		return ErrNotFound
	case mongo.IsDuplicateKeyError(err):
		return ErrDuplicate
	}
	return err
}
