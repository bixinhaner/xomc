package admin

import (
	"context"
	"fmt"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// DictionaryService provides business logic for dictionary management.
type DictionaryService struct {
	dictRepo   DictionaryRepository
	detailRepo DictionaryDetailRepository
}

// NewDictionaryService creates a new DictionaryService.
func NewDictionaryService(dictRepo DictionaryRepository, detailRepo DictionaryDetailRepository) *DictionaryService {
	return &DictionaryService{dictRepo: dictRepo, detailRepo: detailRepo}
}

// CreateDictionary creates a new dictionary entry.
func (s *DictionaryService) CreateDictionary(ctx context.Context, req CreateDictionaryRequest) (*Dictionary, error) {
	dict := &Dictionary{
		Name:        req.Name,
		Type:        req.Type,
		Status:      true,
		Description: req.Description,
	}
	if req.Status != nil {
		dict.Status = *req.Status
	}

	if err := s.dictRepo.Create(ctx, dict); err != nil {
		return nil, fmt.Errorf("create dictionary: %w", err)
	}
	return dict, nil
}

// GetDictionaryByType returns a dictionary with its active details by type.
func (s *DictionaryService) GetDictionaryByType(ctx context.Context, dictType string) (*Dictionary, error) {
	return s.dictRepo.GetByType(ctx, dictType)
}

// ListDictionaries returns all dictionaries (without details).
func (s *DictionaryService) ListDictionaries(ctx context.Context) ([]Dictionary, error) {
	return s.dictRepo.List(ctx)
}

// UpdateDictionary updates an existing dictionary.
func (s *DictionaryService) UpdateDictionary(ctx context.Context, req UpdateDictionaryRequest) (*Dictionary, error) {
	dict, err := s.dictRepo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("get dictionary for update: %w", err)
	}

	if req.Name != nil {
		dict.Name = *req.Name
	}
	if req.Type != nil {
		dict.Type = *req.Type
	}
	if req.Status != nil {
		dict.Status = *req.Status
	}
	if req.Description != nil {
		dict.Description = *req.Description
	}

	if err := s.dictRepo.Update(ctx, dict); err != nil {
		return nil, fmt.Errorf("update dictionary: %w", err)
	}
	return dict, nil
}

// DeleteDictionary soft-deletes a dictionary and its details.
func (s *DictionaryService) DeleteDictionary(ctx context.Context, id int64) error {
	return s.dictRepo.Delete(ctx, id)
}

// CreateDictionaryDetail creates a new dictionary detail entry.
func (s *DictionaryService) CreateDictionaryDetail(ctx context.Context, req CreateDictionaryDetailRequest) (*DictionaryDetail, error) {
	// Verify parent dictionary exists
	if _, err := s.dictRepo.GetByID(ctx, req.SysDictionaryID); err != nil {
		return nil, commonerrors.NewBusinessError(7003, "parent dictionary not found", err)
	}

	detail := &DictionaryDetail{
		Label:           req.Label,
		Value:           req.Value,
		Extend:          req.Extend,
		Status:          true,
		Sort:            0,
		SysDictionaryID: req.SysDictionaryID,
	}
	if req.Status != nil {
		detail.Status = *req.Status
	}
	if req.Sort != nil {
		detail.Sort = *req.Sort
	}

	if err := s.detailRepo.Create(ctx, detail); err != nil {
		return nil, fmt.Errorf("create dictionary detail: %w", err)
	}
	return detail, nil
}

// GetDictionaryDetail returns a dictionary detail by ID.
func (s *DictionaryService) GetDictionaryDetail(ctx context.Context, id int64) (*DictionaryDetail, error) {
	return s.detailRepo.GetByID(ctx, id)
}

// ListDictionaryDetails returns filtered, paginated dictionary details.
func (s *DictionaryService) ListDictionaryDetails(ctx context.Context, req DictionaryDetailListRequest) (*model.ListResponse[DictionaryDetail], error) {
	items, total, err := s.detailRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []DictionaryDetail{}
	}
	return model.NewListResponse(items, total, req.Page, req.PageSize), nil
}

// UpdateDictionaryDetail updates an existing dictionary detail.
func (s *DictionaryService) UpdateDictionaryDetail(ctx context.Context, req UpdateDictionaryDetailRequest) (*DictionaryDetail, error) {
	detail, err := s.detailRepo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("get detail for update: %w", err)
	}

	if req.Label != nil {
		detail.Label = *req.Label
	}
	if req.Value != nil {
		detail.Value = *req.Value
	}
	if req.Extend != nil {
		detail.Extend = *req.Extend
	}
	if req.Status != nil {
		detail.Status = *req.Status
	}
	if req.Sort != nil {
		detail.Sort = *req.Sort
	}
	if req.SysDictionaryID != nil {
		// Verify new parent dictionary exists
		if _, err := s.dictRepo.GetByID(ctx, *req.SysDictionaryID); err != nil {
			return nil, commonerrors.NewBusinessError(7003, "parent dictionary not found", err)
		}
		detail.SysDictionaryID = *req.SysDictionaryID
	}

	if err := s.detailRepo.Update(ctx, detail); err != nil {
		return nil, fmt.Errorf("update dictionary detail: %w", err)
	}
	return detail, nil
}

// DeleteDictionaryDetail soft-deletes a dictionary detail.
func (s *DictionaryService) DeleteDictionaryDetail(ctx context.Context, id int64) error {
	return s.detailRepo.Delete(ctx, id)
}
