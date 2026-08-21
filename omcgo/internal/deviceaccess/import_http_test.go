package deviceaccess

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type importHTTPServiceStub struct {
	actor      PolicyActor
	request    ImportPreviewRequest
	commitID   string
	rollbackID string
}

func (s *importHTTPServiceStub) PreviewAccessList(_ context.Context, actor PolicyActor, request ImportPreviewRequest) (ImportPreviewResult, error) {
	s.actor, s.request = actor, request
	return ImportPreviewResult{Batch: ImportBatch{ID: "batch-1", Carrier: actor.Carrier}}, nil
}
func (s *importHTTPServiceStub) PreviewRuleDimension(_ context.Context, actor PolicyActor, request ImportPreviewRequest) (RuleDimensionPreviewResult, error) {
	s.actor, s.request = actor, request
	return RuleDimensionPreviewResult{Batch: ImportBatch{ID: "batch-1", Carrier: actor.Carrier}}, nil
}
func (s *importHTTPServiceStub) CommitImport(_ context.Context, _ PolicyActor, id string) (ImportBatch, error) {
	s.commitID = id
	return ImportBatch{ID: id}, nil
}
func (s *importHTTPServiceStub) RollbackImport(_ context.Context, _ PolicyActor, id string) (ImportBatch, error) {
	s.rollbackID = id
	return ImportBatch{ID: id}, nil
}
func (s *importHTTPServiceStub) ListImportBatches(_ context.Context, actor PolicyActor, _ ImportBatchListFilter) ([]ImportBatch, int64, error) {
	return []ImportBatch{{ID: "batch-1", Carrier: actor.Carrier}}, 1, nil
}
func (s *importHTTPServiceStub) GetImportBatch(_ context.Context, actor PolicyActor, id string) (ImportBatchDetail, error) {
	return ImportBatchDetail{Batch: ImportBatch{ID: id, Carrier: actor.Carrier}}, nil
}
func (s *importHTTPServiceStub) ListImportErrors(context.Context, PolicyActor, string) ([]ImportRow, error) {
	return []ImportRow{}, nil
}

func makeImportHTTPRequest(t *testing.T, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "../lists.csv")
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	for key, value := range map[string]string{"entry_type": "deny", "mode": "replace", "failure_policy": "strict"} {
		require.NoError(t, writer.WriteField(key, value))
	}
	require.NoError(t, writer.Close())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/device-access/imports/preview", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Idempotency-Key", "preview-1")
	return request
}

func TestPolicyHTTPHandlerImportRoutesUseActorScopeAndSafeFilename(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &importHTTPServiceStub{}
	handler := NewPolicyHTTPHandler(nil, nil, actorResolverStub{actor: PolicyActor{Carrier: "cmcc", SubjectID: "operator-1"}})
	handler.SetImportPreviewService(service)
	handler.SetImportBatchMutationService(service)
	handler.SetImportBatchReadService(service)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))

	response := httptest.NewRecorder()
	router.ServeHTTP(response, makeImportHTTPRequest(t, []byte("csv")))
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "cmcc", service.actor.Carrier)
	require.Equal(t, "lists.csv", service.request.SourceFilename)
	require.Equal(t, "preview-1", service.request.IdempotencyKey)

	for _, endpoint := range []string{"/api/v1/device-access/imports", "/api/v1/device-access/imports/batch-1", "/api/v1/device-access/imports/batch-1/errors"} {
		response = httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, endpoint, nil))
		require.Equal(t, http.StatusOK, response.Code, endpoint)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/device-access/imports/batch-1/commit", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "batch-1", service.commitID)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/device-access/imports/batch-1/rollback", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "batch-1", service.rollbackID)
}

func TestPolicyHTTPHandlerPreviewImportMapsTemplateErrorToBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewPolicyHTTPHandler(nil, nil, actorResolverStub{actor: PolicyActor{Carrier: "cmcc", SubjectID: "operator-1"}})
	handler.SetImportPreviewService(NewImportService(&importStoreStub{}))
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, makeImportHTTPRequest(t, []byte("not,csv")))
	require.Equal(t, http.StatusBadRequest, response.Code)
}
