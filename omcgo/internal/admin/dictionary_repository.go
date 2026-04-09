package admin

import (
	"context"
)

// DictionaryRepository defines the persistence interface for dictionaries.
type DictionaryRepository interface {
	// Create inserts a new dictionary.
	Create(ctx context.Context, dict *Dictionary) error
	// GetByID returns a dictionary by its ID.
	GetByID(ctx context.Context, id int64) (*Dictionary, error)
	// GetByType returns a dictionary with its active details by type.
	GetByType(ctx context.Context, dictType string) (*Dictionary, error)
	// List returns all dictionaries (without details).
	List(ctx context.Context) ([]Dictionary, error)
	// Update updates a dictionary.
	Update(ctx context.Context, dict *Dictionary) error
	// Delete soft-deletes a dictionary and cascades to its details.
	Delete(ctx context.Context, id int64) error
}

// DictionaryDetailRepository defines the persistence interface for dictionary details.
type DictionaryDetailRepository interface {
	// Create inserts a new dictionary detail.
	Create(ctx context.Context, detail *DictionaryDetail) error
	// GetByID returns a detail by its ID.
	GetByID(ctx context.Context, id int64) (*DictionaryDetail, error)
	// List returns filtered, paginated dictionary details.
	List(ctx context.Context, req DictionaryDetailListRequest) ([]DictionaryDetail, int64, error)
	// Update updates a dictionary detail.
	Update(ctx context.Context, detail *DictionaryDetail) error
	// Delete soft-deletes a dictionary detail.
	Delete(ctx context.Context, id int64) error
	// DeleteByDictionaryID soft-deletes all details for a dictionary.
	DeleteByDictionaryID(ctx context.Context, dictID int64) error
}
