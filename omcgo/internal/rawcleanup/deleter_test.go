package rawcleanup

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type fakeObjectStore struct {
	got map[string][]string
	err map[string]map[string]error
}

func (f *fakeObjectStore) DeleteObjects(_ context.Context, bucket string, keys []string) map[string]error {
	f.got[bucket] = append([]string(nil), keys...)
	return f.err[bucket]
}

func TestDeleterUsesExactKeysAndPreservesPartialFailures(t *testing.T) {
	store := &fakeObjectStore{got: map[string][]string{}, err: map[string]map[string]error{
		"pm-files": {"bad.xml": errors.New("throttled")},
	}}
	d := NewDeleter(store, "pm-files", "mr-files")
	in := []Candidate{
		{ID: uuid.New(), Kind: KindPM, ObjectPath: "pm-files/good.xml"},
		{ID: uuid.New(), Kind: KindPM, ObjectPath: "pm-files/bad.xml"},
		{ID: uuid.New(), Kind: KindMR, ObjectPath: "mr-files/mr.xml"},
	}
	got := d.Delete(context.Background(), in)
	if len(store.got["pm-files"]) != 2 || store.got["pm-files"][0] != "good.xml" ||
		len(store.got["mr-files"]) != 1 || store.got["mr-files"][0] != "mr.xml" {
		t.Fatalf("unexpected exact keys: %#v", store.got)
	}
	if len(got) != 3 || got[0].Err != nil || got[1].Err == nil || got[2].Err != nil {
		t.Fatalf("unexpected results: %#v", got)
	}
}
