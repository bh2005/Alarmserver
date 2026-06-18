package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleUpload_StoresFileUnderSoundsDir(t *testing.T) {
	dir := t.TempDir()
	s := NewAnnouncementServer(dir)

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	w.WriteField("name", "evening-tone")
	fw, _ := w.CreateFormFile("file", "ignored.wav")
	fw.Write([]byte("fake wav data"))
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/announcements", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	s.HandleUpload(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	data, err := os.ReadFile(filepath.Join(dir, "evening-tone.wav"))
	if err != nil {
		t.Fatalf("expected file to be stored: %v", err)
	}
	if string(data) != "fake wav data" {
		t.Fatalf("unexpected file contents: %q", data)
	}
}

func TestHandleUpload_RejectsUnsafeName(t *testing.T) {
	dir := t.TempDir()
	s := NewAnnouncementServer(dir)

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	w.WriteField("name", "../../etc/passwd")
	fw, _ := w.CreateFormFile("file", "x.wav")
	fw.Write([]byte("x"))
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/announcements", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	s.HandleUpload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsafe name, got %d", rec.Code)
	}
}

func TestHandleUpload_WrongMethod(t *testing.T) {
	s := NewAnnouncementServer(t.TempDir())
	req := httptest.NewRequest(http.MethodGet, "/api/announcements", nil)
	rec := httptest.NewRecorder()
	s.HandleUpload(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleList_ReturnsUploadedNames(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "alarm-announcement.wav"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "evening-tone.wav"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "not-a-sound.txt"), []byte("x"), 0o644)

	s := NewAnnouncementServer(dir)
	req := httptest.NewRequest(http.MethodGet, "/api/announcements", nil)
	rec := httptest.NewRecorder()
	s.HandleList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var got struct{ Announcements []string `json:"announcements"` }
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	want := []string{"alarm-announcement", "evening-tone"}
	if len(got.Announcements) != len(want) {
		t.Fatalf("expected %v, got %v", want, got.Announcements)
	}
	for i := range want {
		if got.Announcements[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got.Announcements)
		}
	}
}

func TestHandleList_MissingDirReturnsEmptyList(t *testing.T) {
	s := NewAnnouncementServer(filepath.Join(t.TempDir(), "does-not-exist"))
	req := httptest.NewRequest(http.MethodGet, "/api/announcements", nil)
	rec := httptest.NewRecorder()
	s.HandleList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var got struct{ Announcements []string `json:"announcements"` }
	json.Unmarshal(rec.Body.Bytes(), &got)
	if len(got.Announcements) != 0 {
		t.Fatalf("expected empty list, got %v", got.Announcements)
	}
}
