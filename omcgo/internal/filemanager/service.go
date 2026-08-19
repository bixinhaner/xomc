package filemanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/storageprotection"
	"github.com/omcgo/omcgo/internal/task"
)

// FileService provides business logic for file management operations.
type FileService struct {
	repo             FileRepository
	minioClient      *minio.Client
	bucket           string
	taskSvc          task.Enqueuer
	storageAdmission storageprotection.WriteAdmission
	logger           *zap.Logger
}

func (s *FileService) SetStorageAdmission(admission storageprotection.WriteAdmission) {
	s.storageAdmission = admission
}

// NewFileService creates a new FileService.
func NewFileService(
	repo FileRepository,
	minioClient *minio.Client,
	bucket string,
	taskSvc task.Enqueuer,
	logger *zap.Logger,
) *FileService {
	return &FileService{
		repo:        repo,
		minioClient: minioClient,
		bucket:      bucket,
		taskSvc:     taskSvc,
		logger:      logger.Named("filemanager"),
	}
}

// UploadFile stores a file to MinIO and creates a metadata record in the database.
func (s *FileService) UploadFile(ctx context.Context, file io.Reader, fileSize int64, originalName string, contentType string, fileType FileType, description, uploader, deviceSN string) (*ManagedFile, error) {
	// Build MinIO path: managed-files/{file_type}/{date}/{filename}
	objectPath := fmt.Sprintf("managed-files/%s/%s/%s", fileType, time.Now().Format("2006-01-02"), originalName)

	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if s.storageAdmission != nil {
		decision, err := s.storageAdmission.CheckPath(ctx, storageprotection.ProtectedPathIDMinIO, storageprotection.WriteScopeUpload)
		if err != nil {
			return nil, fmt.Errorf("storage admission check: %w", err)
		}
		if !decision.Allowed {
			return nil, fmt.Errorf("storage write protected: %s", decision.Reason)
		}
	}

	// Upload to MinIO
	_, err := s.minioClient.PutObject(ctx, s.bucket, objectPath, file, fileSize,
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return nil, fmt.Errorf("upload to storage: %w", err)
	}

	// Build metadata record
	mf := &ManagedFile{
		FileName:    originalName,
		FileType:    fileType,
		FileSize:    fileSize,
		MinIOPath:   objectPath,
		ContentType: contentType,
		Status:      FileReady,
	}
	if uploader != "" {
		mf.Uploader = &uploader
	}
	if deviceSN != "" {
		mf.DeviceSN = &deviceSN
	}
	if description != "" {
		mf.Description = &description
	}

	if err := s.repo.Create(ctx, mf); err != nil {
		return nil, fmt.Errorf("create file record: %w", err)
	}

	return mf, nil
}

// DownloadFile retrieves a file object from MinIO along with its metadata.
func (s *FileService) DownloadFile(ctx context.Context, id uuid.UUID) (*minio.Object, *ManagedFile, error) {
	mf, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	obj, err := s.minioClient.GetObject(ctx, s.bucket, mf.MinIOPath, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("retrieve from storage: %w", err)
	}

	return obj, mf, nil
}

// DeleteFile removes a file from MinIO and deletes its metadata from the database.
func (s *FileService) DeleteFile(ctx context.Context, id uuid.UUID) error {
	mf, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get file for delete: %w", err)
	}

	// Delete from MinIO — warn on failure but continue with DB deletion
	if err := s.minioClient.RemoveObject(ctx, s.bucket, mf.MinIOPath, minio.RemoveObjectOptions{}); err != nil {
		s.logger.Warn("remove file from MinIO", zap.String("path", mf.MinIOPath), zap.Error(err))
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete file record: %w", err)
	}

	return nil
}

// DistributeFile queues download commands for the given devices.
// Returns (succeeded, failed, error).
func (s *FileService) DistributeFile(ctx context.Context, id uuid.UUID, deviceSNs []string) (int, int, error) {
	if s.taskSvc == nil {
		return 0, 0, fmt.Errorf("task service not configured")
	}

	mf, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return 0, 0, err
	}

	distID := uuid.New().String()
	downloadURL := fmt.Sprintf("%s/%s", s.bucket, mf.MinIOPath)
	tr069FileType := mapFileTypeToTR069(mf.FileType)

	var succeeded, failed int
	for _, deviceSN := range deviceSNs {
		params, _ := json.Marshal(map[string]string{
			"CommandKey":     distID,
			"FileType":       tr069FileType,
			"URL":            downloadURL,
			"TargetFileName": mf.FileName,
		})
		if _, err := s.taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
			DeviceSN:   deviceSN,
			Method:     "Download",
			Params:     params,
			Priority:   5,
			CommandKey: fmt.Sprintf("dist-%s-%s", distID, deviceSN),
			Source:     task.TaskSourceAPI,
		}); err != nil {
			s.logger.Warn("push download command failed",
				zap.String("device_sn", deviceSN),
				zap.Error(err),
			)
			failed++
			continue
		}
		succeeded++
	}

	s.logger.Info("file distribution queued",
		zap.String("dist_id", distID),
		zap.String("file_id", mf.ID.String()),
		zap.Int("succeeded", succeeded),
		zap.Int("failed", failed),
	)

	return succeeded, failed, nil
}

// ListFiles returns a paginated list of files matching the given filter.
func (s *FileService) ListFiles(ctx context.Context, filter FileFilter) (*model.ListResponse[ManagedFile], error) {
	return s.repo.List(ctx, filter)
}

// GetFileByID returns a single file by its ID.
func (s *FileService) GetFileByID(ctx context.Context, id uuid.UUID) (*ManagedFile, error) {
	return s.repo.GetByID(ctx, id)
}

// mapFileTypeToTR069 maps our FileType to TR-069 FileType codes.
func mapFileTypeToTR069(ft FileType) string {
	switch ft {
	case FileFirmware:
		return "1" // Firmware Upgrade Image
	case FileConfig:
		return "3" // Vendor Configuration File
	default:
		return "3" // Default to vendor config
	}
}
