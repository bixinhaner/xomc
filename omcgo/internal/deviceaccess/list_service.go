package deviceaccess

import (
	"context"
	"fmt"
	"strings"
)

type ListEntryStore interface {
	UpsertListEntryAndQueue(ctx context.Context, carrier string, entry CompiledListEntry) error
}

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
	return entry, nil
}
