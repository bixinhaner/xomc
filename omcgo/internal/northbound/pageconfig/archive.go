package pageconfig

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	pathpkg "path"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

const (
	DefaultLocalArchiveBucket        = "northbound"
	defaultLocalArchivePrefix        = ""
	defaultLocalArchiveRetentionDays = 7
)

type LocalArchiveOptions struct {
	Prefix        string
	RetentionDays int
}

type LocalArchiveObject struct {
	Key         string
	Name        string
	Content     []byte
	ContentType string
	Run         FileRun
}

type LocalArchiveResult struct {
	Bucket      string
	ObjectKey   string
	ObjectName  string
	Bytes       int64
	ContentType string
	RunID       string
	ProfileKind ProfileKind
	ProfileCode string
}

type LocalArchiveCleanupRequest struct {
	Prefix string
	Before time.Time
}

type LocalArchiveCleanupSummary struct {
	Bucket         string
	Prefix         string
	Before         time.Time
	RetentionDays  int
	ObjectsDeleted int64
	BytesDeleted   int64
}

type RunArtifactDownload struct {
	FileName    string
	ContentType string
	Content     []byte
}

type LocalArchiveStore interface {
	Put(ctx context.Context, object LocalArchiveObject) (LocalArchiveResult, error)
	Get(ctx context.Context, result LocalArchiveResult) ([]byte, error)
	CleanupBefore(ctx context.Context, req LocalArchiveCleanupRequest) (LocalArchiveCleanupSummary, error)
}

type MinIOLocalArchiveStore struct {
	client *minio.Client
	bucket string
}

func NewMinIOLocalArchiveStore(client *minio.Client, bucket string) *MinIOLocalArchiveStore {
	return &MinIOLocalArchiveStore{
		client: client,
		bucket: strings.TrimSpace(bucket),
	}
}

func (s *MinIOLocalArchiveStore) Ensure(ctx context.Context) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("northbound local archive MinIO client is not configured")
	}
	if s.bucket == "" {
		return fmt.Errorf("northbound local archive bucket is not configured")
	}
	return s.ensureBucket(ctx, s.bucket)
}

func (s *MinIOLocalArchiveStore) Put(ctx context.Context, object LocalArchiveObject) (LocalArchiveResult, error) {
	bucket := ""
	if s != nil {
		bucket = s.bucket
	}
	result := LocalArchiveResult{
		Bucket:      bucket,
		ObjectKey:   object.Key,
		ObjectName:  object.Name,
		Bytes:       int64(len(object.Content)),
		ContentType: object.ContentType,
		RunID:       object.Run.ID,
		ProfileKind: object.Run.ProfileKind,
		ProfileCode: object.Run.ProfileCode,
	}
	if s == nil || s.client == nil {
		return result, fmt.Errorf("northbound local archive MinIO client is not configured")
	}
	if bucket == "" {
		return result, fmt.Errorf("northbound local archive bucket is not configured")
	}
	if err := s.ensureBucket(ctx, bucket); err != nil {
		return result, err
	}
	if strings.TrimSpace(object.Key) == "" {
		return result, fmt.Errorf("northbound local archive object key is empty")
	}
	contentType := strings.TrimSpace(object.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
		result.ContentType = contentType
	}
	_, err := s.client.PutObject(ctx, bucket, object.Key, bytes.NewReader(object.Content), int64(len(object.Content)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return result, fmt.Errorf("put northbound local archive object: %w", err)
	}
	return result, nil
}

func (s *MinIOLocalArchiveStore) ensureBucket(ctx context.Context, bucket string) error {
	exists, err := s.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check northbound local archive bucket: %w", err)
	}
	if exists {
		return nil
	}
	if err := s.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("create northbound local archive bucket: %w", err)
	}
	return nil
}

func (s *MinIOLocalArchiveStore) Get(ctx context.Context, result LocalArchiveResult) ([]byte, error) {
	bucket := strings.TrimSpace(result.Bucket)
	if bucket == "" && s != nil {
		bucket = s.bucket
	}
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("northbound local archive MinIO client is not configured")
	}
	if bucket == "" {
		return nil, fmt.Errorf("northbound local archive bucket is not configured")
	}
	if strings.TrimSpace(result.ObjectKey) == "" {
		return nil, fmt.Errorf("northbound local archive object key is empty")
	}
	obj, err := s.client.GetObject(ctx, bucket, result.ObjectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get northbound local archive object: %w", err)
	}
	defer obj.Close()
	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("read northbound local archive object: %w", err)
	}
	return data, nil
}

func (s *MinIOLocalArchiveStore) CleanupBefore(ctx context.Context, req LocalArchiveCleanupRequest) (LocalArchiveCleanupSummary, error) {
	bucket := ""
	if s != nil {
		bucket = s.bucket
	}
	summary := LocalArchiveCleanupSummary{
		Bucket: bucket,
		Prefix: cleanLocalArchivePrefix(req.Prefix),
		Before: req.Before,
	}
	if s == nil || s.client == nil {
		return summary, fmt.Errorf("northbound local archive MinIO client is not configured")
	}
	if bucket == "" {
		return summary, fmt.Errorf("northbound local archive bucket is not configured")
	}
	if req.Before.IsZero() {
		return summary, fmt.Errorf("northbound local archive cleanup cutoff is empty")
	}
	prefix := summary.Prefix
	if prefix != "" {
		prefix += "/"
	}
	objectCh := s.client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})
	for object := range objectCh {
		if object.Err != nil {
			return summary, fmt.Errorf("list northbound local archive objects: %w", object.Err)
		}
		if object.Key == "" || object.LastModified.IsZero() || !object.LastModified.Before(req.Before) {
			continue
		}
		if err := s.client.RemoveObject(ctx, bucket, object.Key, minio.RemoveObjectOptions{}); err != nil {
			return summary, fmt.Errorf("remove northbound local archive object %s: %w", object.Key, err)
		}
		summary.ObjectsDeleted++
		summary.BytesDeleted += object.Size
	}
	return summary, nil
}

func (s *Service) archiveRunLocally(ctx context.Context, run FileRun) (*LocalArchiveResult, error) {
	if s.localArchive == nil || run.Status != RunStatusSuccess || strings.TrimSpace(run.ArtifactContent) == "" {
		return nil, nil
	}
	artifact, err := materializeRunArtifact(run)
	if err != nil {
		_ = s.createLocalArchiveEvent(ctx, run, LocalArchiveResult{}, err)
		return nil, err
	}
	opts := normalizeLocalArchiveOptions(s.localArchiveOptions)
	object := LocalArchiveObject{
		Key:         localArchiveObjectKey(opts.Prefix, run),
		Name:        run.ArtifactName,
		Content:     artifact,
		ContentType: localArchiveContentType(run.ArtifactName),
		Run:         run,
	}
	result, err := s.localArchive.Put(ctx, object)
	_ = s.createLocalArchiveEvent(ctx, run, result, err)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Service) markRunArchived(ctx context.Context, run *FileRun, result *LocalArchiveResult) error {
	if s.repo == nil || run == nil || result == nil {
		return nil
	}
	updated, err := s.repo.UpdateFileRunLocalArchive(ctx, run.ID, *result, true)
	if err != nil {
		return err
	}
	*run = *updated
	return nil
}

func (s *Service) DownloadRunArtifact(ctx context.Context, id string) (RunArtifactDownload, error) {
	run, err := s.GetRun(ctx, id)
	if err != nil {
		return RunArtifactDownload{}, err
	}
	if run.Status != RunStatusSuccess {
		return RunArtifactDownload{}, fmt.Errorf("%w: northbound run artifact is not available", commonerrors.ErrInvalidInput)
	}
	fileName := run.ArtifactName
	if strings.TrimSpace(fileName) == "" {
		fileName = "northbound-artifact.txt"
	}
	if strings.TrimSpace(run.ArtifactContent) != "" {
		artifact, err := materializeRunArtifact(*run)
		if err != nil {
			return RunArtifactDownload{}, err
		}
		return RunArtifactDownload{
			FileName:    fileName,
			ContentType: contentTypeForArtifact(fileName),
			Content:     artifact,
		}, nil
	}
	if s.localArchive == nil {
		return RunArtifactDownload{}, fmt.Errorf("%w: northbound run artifact is not available", commonerrors.ErrInvalidInput)
	}
	ref, ok := localArchiveResultFromRun(*run)
	if !ok {
		return RunArtifactDownload{}, fmt.Errorf("%w: northbound run artifact is not available", commonerrors.ErrInvalidInput)
	}
	artifact, err := s.localArchive.Get(ctx, ref)
	if err != nil {
		return RunArtifactDownload{}, err
	}
	return RunArtifactDownload{
		FileName:    firstNonEmpty(ref.ObjectName, fileName),
		ContentType: firstNonEmpty(ref.ContentType, contentTypeForArtifact(fileName)),
		Content:     artifact,
	}, nil
}

// hydrateRunArtifactContent restores the preview text from the local archive
// without putting the artifact back into northbound_file_runs. The database
// keeps metadata only after a successful MinIO archive; the page-config API
// can still show the actual CSV/XML content on demand.
func (s *Service) hydrateRunArtifactContent(ctx context.Context, run *FileRun) error {
	if s == nil || s.localArchive == nil || run == nil || run.Status != RunStatusSuccess || strings.TrimSpace(run.ArtifactContent) != "" {
		return nil
	}
	ref, ok := localArchiveResultFromRun(*run)
	if !ok {
		return nil
	}
	artifact, err := s.localArchive.Get(ctx, ref)
	if err != nil {
		return err
	}
	content, err := unmaterializeRunArtifact(*run, artifact)
	if err != nil {
		return err
	}
	run.ArtifactContent = string(content)
	return nil
}

func unmaterializeRunArtifact(run FileRun, artifact []byte) ([]byte, error) {
	if !run.CompressionEnabled {
		return artifact, nil
	}
	switch compressionFormatOrDefault(run.CompressionFormat) {
	case CompressionGz:
		reader, err := gzip.NewReader(bytes.NewReader(artifact))
		if err != nil {
			return nil, fmt.Errorf("open gzip northbound artifact: %w", err)
		}
		defer reader.Close()
		content, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("read gzip northbound artifact: %w", err)
		}
		return content, nil
	default:
		reader, err := zip.NewReader(bytes.NewReader(artifact), int64(len(artifact)))
		if err != nil {
			return nil, fmt.Errorf("open zip northbound artifact: %w", err)
		}
		if len(reader.File) == 0 {
			return nil, fmt.Errorf("zip northbound artifact has no files")
		}
		file, err := reader.File[0].Open()
		if err != nil {
			return nil, fmt.Errorf("open file in zip northbound artifact: %w", err)
		}
		defer file.Close()
		content, err := io.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("read file in zip northbound artifact: %w", err)
		}
		return content, nil
	}
}

func (s *Service) createLocalArchiveEvent(ctx context.Context, run FileRun, result LocalArchiveResult, archiveErr error) error {
	if s.repo == nil {
		return nil
	}
	_, err := s.repo.CreateEvent(ctx, eventFromLocalArchiveRun(run, result, archiveErr))
	return err
}

func normalizeLocalArchiveOptions(opts LocalArchiveOptions) LocalArchiveOptions {
	opts.Prefix = cleanLocalArchivePrefix(opts.Prefix)
	if opts.Prefix == "" || opts.Prefix == "." || strings.Contains(opts.Prefix, "\x00") {
		opts.Prefix = defaultLocalArchivePrefix
	}
	if opts.RetentionDays <= 0 {
		opts.RetentionDays = defaultLocalArchiveRetentionDays
	}
	return opts
}

func localArchiveObjectKey(prefix string, run FileRun) string {
	createdAt := run.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	parts := []string{
		cleanLocalArchivePrefix(prefix),
		createdAt.Local().Format("2006-01-02"),
		safeLocalArchiveSegment(run.ProfileCode),
	}
	if groupID := strings.TrimSpace(run.GroupID); groupID != "" && !strings.EqualFold(groupID, strings.TrimSpace(run.ProfileCode)) {
		parts = append(parts, safeLocalArchiveSegment(groupID))
	}
	if tech := runTechnologyDirectory(run); tech != "" {
		parts = append(parts, safeLocalArchiveSegment(tech))
	}
	parts = append(parts, safeRemoteBase(run.ArtifactName))
	return pathpkg.Join(parts...)
}

func cleanLocalArchivePrefix(prefix string) string {
	prefix = strings.TrimSpace(strings.ReplaceAll(prefix, "\\", "/"))
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return ""
	}
	cleaned := pathpkg.Clean(prefix)
	if cleaned == "." || strings.Contains(cleaned, "\x00") {
		return ""
	}
	return cleaned
}

func safeLocalArchiveSegment(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	value = pathpkg.Base(value)
	if value == "." || value == "/" || value == "" || strings.Contains(value, "\x00") {
		return "_"
	}
	return value
}

func localArchiveContentType(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.HasSuffix(name, ".zip"), strings.HasSuffix(name, ".gz"):
		return "application/octet-stream"
	case strings.HasSuffix(name, ".csv"):
		return "text/csv; charset=utf-8"
	case strings.HasSuffix(name, ".xml"):
		return "application/xml; charset=utf-8"
	case strings.HasSuffix(name, ".txt"):
		return "text/plain; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}

func localArchiveResultFromRun(run FileRun) (LocalArchiveResult, bool) {
	raw, ok := run.Summary["local_archive"]
	if !ok {
		return LocalArchiveResult{}, false
	}
	values, ok := raw.(map[string]any)
	if !ok {
		return LocalArchiveResult{}, false
	}
	result := LocalArchiveResult{
		Bucket:      stringFromSummary(values["bucket"]),
		ObjectKey:   stringFromSummary(values["object_key"]),
		ObjectName:  stringFromSummary(values["object_name"]),
		Bytes:       int64FromSummary(values["bytes"]),
		ContentType: stringFromSummary(values["content_type"]),
		RunID:       run.ID,
		ProfileKind: run.ProfileKind,
		ProfileCode: run.ProfileCode,
	}
	if result.ObjectKey == "" {
		return LocalArchiveResult{}, false
	}
	return result, true
}

func stringFromSummary(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func int64FromSummary(value any) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}

type localArchivePayload struct {
	Bucket      string      `json:"bucket"`
	ObjectKey   string      `json:"object_key"`
	ObjectName  string      `json:"object_name"`
	Bytes       int64       `json:"bytes"`
	ContentType string      `json:"content_type"`
	RunID       string      `json:"run_id"`
	ProfileKind ProfileKind `json:"profile_kind"`
	ProfileCode string      `json:"profile_code"`
}

func eventFromLocalArchiveRun(run FileRun, result LocalArchiveResult, archiveErr error) PageConfigEvent {
	if result.RunID == "" {
		result.RunID = run.ID
	}
	if result.ProfileKind == "" {
		result.ProfileKind = run.ProfileKind
	}
	if result.ProfileCode == "" {
		result.ProfileCode = run.ProfileCode
	}
	status := RunStatusSuccess
	errorMessage := ""
	if archiveErr != nil {
		status = RunStatusFailed
		errorMessage = archiveErr.Error()
	}
	payload := mustMarshalEventPayload(localArchivePayload{
		Bucket:      result.Bucket,
		ObjectKey:   result.ObjectKey,
		ObjectName:  result.ObjectName,
		Bytes:       result.Bytes,
		ContentType: result.ContentType,
		RunID:       result.RunID,
		ProfileKind: result.ProfileKind,
		ProfileCode: result.ProfileCode,
	})
	artifactPath := result.ObjectKey
	if result.Bucket != "" && result.ObjectKey != "" {
		artifactPath = result.Bucket + "/" + result.ObjectKey
	}
	return PageConfigEvent{
		Capability:         firstNonEmpty(string(run.ProfileKind), string(ProfileKindFile)),
		OwnerCode:          run.ProfileCode,
		TargetKey:          "minio",
		EventType:          "local_archive",
		Status:             status,
		ArtifactType:       EventArtifactJSON,
		ArtifactName:       firstNonEmpty(result.ObjectName, run.ArtifactName),
		ArtifactPath:       artifactPath,
		Payload:            payload,
		PayloadContentType: "application/json; charset=utf-8",
		ErrorMessage:       errorMessage,
		Summary: map[string]any{
			"run_id":               run.ID,
			"profile_kind":         run.ProfileKind,
			"profile_code":         run.ProfileCode,
			"group_id":             run.GroupID,
			"domain":               run.Domain,
			"object_code":          run.ObjectCode,
			"technology":           run.Summary["technology"],
			"technology_label":     run.Summary["technology_label"],
			"technology_directory": run.Summary["technology_directory"],
			"bucket":               result.Bucket,
			"object_key":           result.ObjectKey,
			"bytes":                result.Bytes,
			"content_type":         result.ContentType,
		},
	}
}
