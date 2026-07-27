package rawcleanup

import (
	"context"
	"strings"

	"github.com/minio/minio-go/v7"
)

type ObjectStore interface {
	DeleteObjects(context.Context, string, []string) map[string]error
}

type Deleter struct {
	store              ObjectStore
	pmBucket, mrBucket string
}

func NewDeleter(store ObjectStore, pmBucket, mrBucket string) *Deleter {
	return &Deleter{store: store, pmBucket: pmBucket, mrBucket: mrBucket}
}

func (d *Deleter) Delete(ctx context.Context, candidates []Candidate) []DeleteResult {
	results := make([]DeleteResult, len(candidates))
	byBucket := map[string][]string{}
	for i, candidate := range candidates {
		results[i].Candidate = candidate
		bucket := d.pmBucket
		if candidate.Kind == KindMR {
			bucket = d.mrBucket
		}
		key := strings.TrimPrefix(candidate.ObjectPath, bucket+"/")
		byBucket[bucket] = append(byBucket[bucket], key)
	}
	failures := map[string]map[string]error{}
	for bucket, keys := range byBucket {
		failures[bucket] = d.store.DeleteObjects(ctx, bucket, keys)
	}
	for i, candidate := range candidates {
		bucket := d.pmBucket
		if candidate.Kind == KindMR {
			bucket = d.mrBucket
		}
		key := strings.TrimPrefix(candidate.ObjectPath, bucket+"/")
		results[i].Err = failures[bucket][key]
	}
	return results
}

type MinIOStore struct{ Client *minio.Client }

func (s MinIOStore) DeleteObjects(ctx context.Context, bucket string, keys []string) map[string]error {
	failures := make(map[string]error)
	objects := make(chan minio.ObjectInfo, len(keys))
	for _, key := range keys {
		objects <- minio.ObjectInfo{Key: key}
	}
	close(objects)
	for removal := range s.Client.RemoveObjects(ctx, bucket, objects, minio.RemoveObjectsOptions{}) {
		if removal.Err != nil && minio.ToErrorResponse(removal.Err).Code != "NoSuchKey" {
			failures[removal.ObjectName] = removal.Err
		}
	}
	// A timed-out batch may not report one result per submitted object. Retrying
	// every key is safe because exact object deletion is idempotent.
	if err := ctx.Err(); err != nil {
		for _, key := range keys {
			if _, failed := failures[key]; !failed {
				failures[key] = err
			}
		}
	}
	return failures
}
