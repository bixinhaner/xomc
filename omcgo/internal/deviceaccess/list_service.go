package deviceaccess

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type ListEntryStore interface {
	UpsertListEntryAndQueue(ctx context.Context, carrier string, entry CompiledListEntry) error
}

type ListEntryBatchStore interface {
	UpsertListEntriesAndQueue(ctx context.Context, carrier string, entries []CompiledListEntry) error
	DisableListEntriesAndQueue(ctx context.Context, carrier string, entryType ListEntryType, serialNumbers []string, reason string) error
}

var ErrListBatchConflict = errors.New("access list batch state conflict")

type ListService struct {
	entries    ListEntryStore
	identities IdentityVisibilityChecker
}

func NewListService(entries ListEntryStore) *ListService {
	return &ListService{entries: entries}
}

func (s *ListService) SetIdentityVisibilityChecker(checker IdentityVisibilityChecker) {
	s.identities = checker
}

// UpsertAndReevaluate persists one list change and its targeted reevaluation in
// the same transaction.
func (s *ListService) UpsertAndReevaluate(
	ctx context.Context,
	actor PolicyActor,
	entry CompiledListEntry,
) error {
	if err := validatePolicyActor(actor); err != nil {
		return err
	}
	if s == nil || s.entries == nil {
		return fmt.Errorf("upsert access list: %w", ErrAccessGateDependencyMissing)
	}
	entry, err := normalizeListEntry(entry)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidAccessInput, err)
	}
	if err := authorizeIdentityScope(ctx, s.identities, actor, entry.IdentityValue); err != nil {
		return err
	}
	if err := s.entries.UpsertListEntryAndQueue(ctx, actor.Carrier, entry); err != nil {
		return fmt.Errorf("upsert access list entry: %w", err)
	}
	return nil
}

// UpsertManyAndReevaluate validates the whole request before asking the store
// to persist it atomically. A failure must not leave a partial access list.
func (s *ListService) UpsertManyAndReevaluate(
	ctx context.Context,
	actor PolicyActor,
	entries []CompiledListEntry,
) error {
	if err := validatePolicyActor(actor); err != nil {
		return err
	}
	if s == nil || s.entries == nil {
		return fmt.Errorf("upsert access list batch: %w", ErrAccessGateDependencyMissing)
	}
	if len(entries) == 0 || len(entries) > 10000 {
		return fmt.Errorf("%w: access list batch must contain between 1 and 10000 entries", ErrInvalidAccessInput)
	}
	batchStore, ok := s.entries.(ListEntryBatchStore)
	if !ok {
		return fmt.Errorf("upsert access list batch: %w", ErrAccessGateDependencyMissing)
	}
	normalized := make([]CompiledListEntry, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	var entryType ListEntryType
	for _, entry := range entries {
		item, err := normalizeListEntry(entry)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidAccessInput, err)
		}
		if entryType == "" {
			entryType = item.Type
		} else if item.Type != entryType {
			return fmt.Errorf("%w: access list batch cannot mix entry types", ErrInvalidAccessInput)
		}
		key := string(item.Type) + "\x00" + item.IdentityValue
		if _, exists := seen[key]; exists {
			return fmt.Errorf("%w: duplicate serial number %q", ErrInvalidAccessInput, item.IdentityValue)
		}
		seen[key] = struct{}{}
		if err := authorizeIdentityScope(ctx, s.identities, actor, item.IdentityValue); err != nil {
			return err
		}
		normalized = append(normalized, item)
	}
	if err := batchStore.UpsertListEntriesAndQueue(ctx, actor.Carrier, normalized); err != nil {
		return fmt.Errorf("upsert access list batch: %w", err)
	}
	return nil
}

func (s *ListService) DisableManyAndReevaluate(
	ctx context.Context,
	actor PolicyActor,
	entryType ListEntryType,
	serialNumbers []string,
	reason string,
) error {
	if err := validatePolicyActor(actor); err != nil {
		return err
	}
	if s == nil || s.entries == nil {
		return fmt.Errorf("disable access list batch: %w", ErrAccessGateDependencyMissing)
	}
	if entryType != ListEntryTypeDeny && entryType != ListEntryTypeAllow && entryType != ListEntryTypeRevoked {
		return fmt.Errorf("%w: unsupported access list type %q", ErrInvalidAccessInput, entryType)
	}
	if len(serialNumbers) == 0 || len(serialNumbers) > 10000 || strings.TrimSpace(reason) == "" {
		return fmt.Errorf("%w: serial numbers and reason are required", ErrInvalidAccessInput)
	}
	batchStore, ok := s.entries.(ListEntryBatchStore)
	if !ok {
		return fmt.Errorf("disable access list batch: %w", ErrAccessGateDependencyMissing)
	}
	normalized := make([]string, 0, len(serialNumbers))
	seen := make(map[string]struct{}, len(serialNumbers))
	for _, value := range serialNumbers {
		serialNumber := strings.TrimSpace(value)
		if serialNumber == "" {
			return fmt.Errorf("%w: serial number is required", ErrInvalidAccessInput)
		}
		if _, exists := seen[serialNumber]; exists {
			return fmt.Errorf("%w: duplicate serial number %q", ErrInvalidAccessInput, serialNumber)
		}
		seen[serialNumber] = struct{}{}
		if err := authorizeIdentityScope(ctx, s.identities, actor, serialNumber); err != nil {
			return err
		}
		normalized = append(normalized, serialNumber)
	}
	if err := batchStore.DisableListEntriesAndQueue(ctx, actor.Carrier, entryType, normalized, strings.TrimSpace(reason)); err != nil {
		return fmt.Errorf("disable access list batch: %w", err)
	}
	return nil
}

func normalizeListEntry(entry CompiledListEntry) (CompiledListEntry, error) {
	if entry.IdentityType != IdentityTypeSerialNumber || strings.TrimSpace(entry.IdentityValue) == "" {
		return CompiledListEntry{}, fmt.Errorf("upsert access list: serial number identity is required")
	}
	switch entry.Type {
	case ListEntryTypeDeny, ListEntryTypeAllow, ListEntryTypeRevoked:
	default:
		return CompiledListEntry{}, fmt.Errorf("upsert access list: unsupported entry type %q", entry.Type)
	}
	if entry.Status == "" {
		entry.Status = ListEntryStatusActive
	}
	if entry.Status != ListEntryStatusActive && entry.Status != ListEntryStatusDisabled {
		return CompiledListEntry{}, fmt.Errorf("upsert access list: unsupported status %q", entry.Status)
	}
	entry.IdentityValue = strings.TrimSpace(entry.IdentityValue)
	entry.Reason = strings.TrimSpace(entry.Reason)
	if entry.Reason == "" {
		entry.Reason = "manual_list_change"
	}
	return entry, nil
}
