package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newRouter(middlewares ...gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Use(middlewares...)
	return r
}

// --- RequestID ---

func TestRequestID(t *testing.T) {
	r := newRouter(RequestID())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, GetRequestID(c))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	header := w.Header().Get("X-Request-ID")
	if header == "" {
		t.Fatal("X-Request-ID header not set")
	}
	if _, err := uuid.Parse(header); err != nil {
		t.Errorf("X-Request-ID is not a valid UUID: %q", header)
	}
	if body := w.Body.String(); body != header {
		t.Errorf("context request_id %q does not match header %q", body, header)
	}
}

func TestRequestID_UniquePerRequest(t *testing.T) {
	r := newRouter(RequestID())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	ids := make(map[string]struct{})
	for range 5 {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		id := w.Header().Get("X-Request-ID")
		if _, seen := ids[id]; seen {
			t.Fatalf("duplicate request ID generated: %q", id)
		}
		ids[id] = struct{}{}
	}
}

func TestGetRequestID_MissingMiddleware(t *testing.T) {
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, GetRequestID(c))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if body := w.Body.String(); body != "" {
		t.Errorf("expected empty string when middleware absent, got %q", body)
	}
}

// --- Logger ---

func TestLogger_LogsRequestFields(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	r := newRouter(RequestID(), NewLogger(logger))
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ping", nil))

	output := buf.String()
	for _, want := range []string{"method=GET", "path=/ping", "status=200", "duration=", "request_id="} {
		if !strings.Contains(output, want) {
			t.Errorf("log missing %q — got: %s", want, output)
		}
	}
}

func TestLogger_PassesThrough(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	r := newRouter(RequestID(), NewLogger(logger))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusNoContent {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNoContent)
	}
}

// --- Recovery ---

func TestRecovery_PanicReturns500(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	r := newRouter(RequestID(), NewLogger(logger), NewRecovery(logger))
	r.GET("/boom", func(c *gin.Context) { panic("something went wrong") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want 500", w.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("parsing response body: %v", err)
	}
	if body["detail"] != "internal server error" {
		t.Errorf("detail: got %q, want %q", body["detail"], "internal server error")
	}
}

func TestRecovery_LogsPanicWithRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	r := newRouter(RequestID(), NewLogger(logger), NewRecovery(logger))
	r.GET("/boom", func(c *gin.Context) { panic("kaboom") })

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/boom", nil))

	output := buf.String()
	if !strings.Contains(output, "panic recovered") {
		t.Error("expected panic recovered log entry")
	}
	if !strings.Contains(output, "request_id=") {
		t.Error("expected request_id in panic log")
	}
}

func TestRecovery_PanicAfterWriteDoesNotDoubleWrite(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	r := newRouter(RequestID(), NewLogger(logger), NewRecovery(logger))
	r.GET("/partial", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"partial": true})
		panic("panic after write")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/partial", nil))

	body := w.Body.String()
	if strings.Count(body, "{") != 1 {
		t.Errorf("response body written twice — got: %s", body)
	}
}

func TestRecovery_NormalRequestPassesThrough(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	r := newRouter(RequestID(), NewLogger(logger), NewRecovery(logger))
	r.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ok", nil))

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}
}
