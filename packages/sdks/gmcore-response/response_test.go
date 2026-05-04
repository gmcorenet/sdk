package gmcore_response

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestJSONPrettyAndError(t *testing.T) {
	recorder := httptest.NewRecorder()
	if err := JSONPretty(recorder, http.StatusCreated, map[string]string{"status": "ok"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(recorder.Body.String(), "\n  ") {
		t.Fatalf("expected pretty json, got %q", recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	if err := Error(recorder, http.StatusBadRequest, "invalid"); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"message":"invalid"`) {
		t.Fatalf("unexpected error payload: code=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestETagAndDeleteCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("If-None-Match", `"abc"`)
	recorder := httptest.NewRecorder()
	if !ETag(recorder, req, `"abc"`) || recorder.Code != http.StatusNotModified {
		t.Fatalf("expected not modified response")
	}
	recorder = httptest.NewRecorder()
	DeleteCookie(recorder, "session", "/")
	if !strings.Contains(recorder.Header().Get("Set-Cookie"), "session=") {
		t.Fatalf("expected deleted cookie header")
	}
}

func TestProblemLastModifiedAndDownloadFile(t *testing.T) {
	recorder := httptest.NewRecorder()
	if err := Problem(recorder, http.StatusUnauthorized, "Unauthorized", "login required"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(recorder.Header().Get("Content-Type"), "application/problem+json") {
		t.Fatalf("unexpected problem content type: %s", recorder.Header().Get("Content-Type"))
	}

	req := httptest.NewRequest("GET", "/", nil)
	modifiedAt := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	req.Header.Set("If-Modified-Since", modifiedAt.Format(http.TimeFormat))
	recorder = httptest.NewRecorder()
	if !LastModified(recorder, req, modifiedAt) {
		t.Fatal("expected not modified")
	}

	root := t.TempDir()
	filePath := filepath.Join(root, "file.txt")
	if err := os.WriteFile(filePath, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	recorder = httptest.NewRecorder()
	if err := DownloadFile(recorder, filePath); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(recorder.Header().Get("Content-Disposition"), "file.txt") {
		t.Fatalf("unexpected download headers: %#v", recorder.Header())
	}
}
