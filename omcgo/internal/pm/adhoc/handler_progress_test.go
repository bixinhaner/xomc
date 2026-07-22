package adhoc

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerProgress_StreamsConnectedProgressAndCompletedThenCleansUp(t *testing.T) {
	taskID := uuid.New()
	repo := &handlerStubRepo{get: func(uuid.UUID) (*Task, error) {
		return &Task{ID: taskID, Creator: "alice", Visibility: VisibilityPrivate}, nil
	}}
	hub := NewProgressHub()
	handler := NewHandler(repo, nil, hub, nil)
	server := httptest.NewServer(progressTestRouter(handler, "alice", false))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		server.URL+"/pm/adhoc/tasks/"+taskID.String()+"/progress", nil)
	require.NoError(t, err)
	resp, err := server.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
	reader := bufio.NewReader(resp.Body)
	require.Equal(t, ":connected\n\n", readSSEFrame(t, reader))

	hub.Publish(taskID.String(), ProgressEvent{
		Name: "progress",
		Data: []byte(`{"task_id":"` + taskID.String() + `","progress":35}`),
	})
	progressFrame := readSSEFrame(t, reader)
	assert.Contains(t, progressFrame, "event: progress\n")
	assert.Contains(t, progressFrame, `data: {"task_id":"`+taskID.String()+`","progress":35}`)

	hub.Publish(taskID.String(), ProgressEvent{
		Name: "completed",
		Data: []byte(`{"task_id":"` + taskID.String() + `","status":"succeeded","rows_total":42}`),
	})
	completedFrame := readSSEFrame(t, reader)
	assert.Contains(t, completedFrame, "event: completed\n")
	assert.Contains(t, completedFrame, `"rows_total":42`)
	_, err = reader.ReadByte()
	assert.ErrorIs(t, err, io.EOF)
}

func TestHandlerProgress_ClientDisconnectCleansUpWithoutAffectingOtherConnection(t *testing.T) {
	taskID := uuid.New()
	repo := &handlerStubRepo{get: func(uuid.UUID) (*Task, error) {
		return &Task{ID: taskID, Creator: "alice", Visibility: VisibilityPrivate}, nil
	}}
	hub := NewProgressHub()
	handler := NewHandler(repo, nil, hub, nil)
	handlerDone := make(chan struct{}, 2)
	server := httptest.NewServer(progressTestRouterWithDone(handler, "alice", false, handlerDone))
	defer server.Close()

	firstCtx, cancelFirst := context.WithCancel(context.Background())
	firstResp := openProgressStream(t, server, firstCtx, taskID)
	defer firstResp.Body.Close()
	secondCtx, cancelSecond := context.WithCancel(context.Background())
	defer cancelSecond()
	secondResp := openProgressStream(t, server, secondCtx, taskID)
	defer secondResp.Body.Close()

	cancelFirst()
	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("first progress handler did not return after client disconnect")
	}
	hub.Publish(taskID.String(), ProgressEvent{Name: "progress", Data: []byte(`{"progress":50}`)})
	frame := readSSEFrame(t, bufio.NewReader(secondResp.Body))
	assert.Contains(t, frame, "event: progress")
}

func TestHandlerProgress_NotFoundBeforeProgressHubSubscription(t *testing.T) {
	repo := &handlerStubRepo{get: func(uuid.UUID) (*Task, error) { return nil, ErrNotFound }}
	hub := NewProgressHub()
	handler := NewHandler(repo, nil, hub, nil)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pm/adhoc/tasks/"+uuid.NewString()+"/progress", nil)
	progressTestRouter(handler, "alice", false).ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

type failingBusinessSSEWriter struct {
	header  http.Header
	err     error
	entered chan struct{}
}

type shortConnectedSSEWriter struct {
	header  http.Header
	entered chan struct{}
	release chan struct{}
}

func (w *shortConnectedSSEWriter) Header() http.Header { return w.header }
func (w *shortConnectedSSEWriter) WriteHeader(int)     {}
func (w *shortConnectedSSEWriter) Flush()              {}
func (w *shortConnectedSSEWriter) FlushError() error   { return nil }
func (w *shortConnectedSSEWriter) Write(data []byte) (int, error) {
	if string(data) == ":connected\n\n" {
		close(w.entered)
		<-w.release
		return len(data) - 1, nil
	}
	return len(data), nil
}

type shortNilErrorWriter struct{}

func (shortNilErrorWriter) Write(data []byte) (int, error) { return len(data) - 1, nil }

func (w *failingBusinessSSEWriter) Header() http.Header { return w.header }
func (w *failingBusinessSSEWriter) WriteHeader(int)     {}
func (w *failingBusinessSSEWriter) Flush()              {}
func (w *failingBusinessSSEWriter) FlushError() error   { return nil }
func (w *failingBusinessSSEWriter) Write(data []byte) (int, error) {
	if string(data) == ":connected\n\n" {
		close(w.entered)
	}
	if strings.HasPrefix(string(data), "event: ") {
		return 0, w.err
	}
	return len(data), nil
}

func TestHandlerProgress_BusinessWriteFailureReturnsAndUnsubscribes(t *testing.T) {
	taskID := uuid.New()
	repo := &handlerStubRepo{get: func(uuid.UUID) (*Task, error) {
		return &Task{ID: taskID, Creator: "alice", Visibility: VisibilityPrivate}, nil
	}}
	hub := NewProgressHub()
	handler := NewHandler(repo, nil, hub, nil)
	writeErr := errors.New("client connection closed")
	writer := &failingBusinessSSEWriter{header: make(http.Header), err: writeErr, entered: make(chan struct{})}
	ctx, _ := gin.CreateTestContext(writer)
	ctx.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	ctx.Set("username", "alice")
	ctx.Request = httptest.NewRequest(http.MethodGet, "/pm/adhoc/tasks/"+taskID.String()+"/progress", nil)

	done := make(chan struct{})
	go func() {
		handler.Progress(ctx)
		close(done)
	}()
	select {
	case <-writer.entered:
	case <-time.After(time.Second):
		t.Fatal("handler did not establish progress stream")
	}
	hub.Publish(taskID.String(), ProgressEvent{Name: "progress", Data: []byte(`{"progress":50}`)})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler did not return after SSE write failure")
	}
}

func TestWriteSSEHelpers_ReturnShortWriteWhenWriterReturnsNoError(t *testing.T) {
	assert.ErrorIs(t, writeSSEString(shortNilErrorWriter{}, "connected"), io.ErrShortWrite)
	assert.ErrorIs(t, writeSSEBytes(shortNilErrorWriter{}, []byte("payload")), io.ErrShortWrite)
}

func TestHandlerProgress_ConnectedShortWriteReturnsAndUnsubscribes(t *testing.T) {
	taskID := uuid.New()
	repo := &handlerStubRepo{get: func(uuid.UUID) (*Task, error) {
		return &Task{ID: taskID, Creator: "alice", Visibility: VisibilityPrivate}, nil
	}}
	hub := NewProgressHub()
	handler := NewHandler(repo, nil, hub, nil)
	writer := &shortConnectedSSEWriter{
		header:  make(http.Header),
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
	ctx, _ := gin.CreateTestContext(writer)
	ctx.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	ctx.Set("username", "alice")
	ctx.Request = httptest.NewRequest(http.MethodGet, "/pm/adhoc/tasks/"+taskID.String()+"/progress", nil)

	done := make(chan struct{})
	go func() {
		handler.Progress(ctx)
		close(done)
	}()
	select {
	case <-writer.entered:
	case <-time.After(time.Second):
		t.Fatal("handler did not attempt connected frame")
	}
	close(writer.release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler did not return after connected short write")
	}
}

func progressTestRouter(handler *Handler, username string, admin bool) http.Handler {
	return newProgressTestRouter(handler, username, admin, nil)
}

func progressTestRouterWithDone(handler *Handler, username string, admin bool, done chan<- struct{}) http.Handler {
	return newProgressTestRouter(handler, username, admin, done)
}

func newProgressTestRouter(handler *Handler, username string, admin bool, done chan<- struct{}) http.Handler {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("username", username)
		c.Set("is_super_admin", admin)
		c.Next()
	})
	if done != nil {
		router.Use(func(c *gin.Context) {
			c.Next()
			done <- struct{}{}
		})
	}
	handler.RegisterRoutes(router.Group(""))
	return router
}

func openProgressStream(t *testing.T, server *httptest.Server, ctx context.Context, taskID uuid.UUID) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		server.URL+"/pm/adhoc/tasks/"+taskID.String()+"/progress", nil)
	require.NoError(t, err)
	resp, err := server.Client().Do(req)
	require.NoError(t, err)
	require.Equal(t, ":connected\n\n", readSSEFrame(t, bufio.NewReader(resp.Body)))
	return resp
}

func readSSEFrame(t *testing.T, reader *bufio.Reader) string {
	t.Helper()
	var frame strings.Builder
	for {
		line, err := reader.ReadString('\n')
		require.NoError(t, err)
		frame.WriteString(line)
		if line == "\n" {
			return frame.String()
		}
	}
}
