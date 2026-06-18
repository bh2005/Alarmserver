package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

var safeAnnouncementName = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// AnnouncementServer manages pre-recorded announcement sound files for the
// voice channel's static (non-TTS) Playback. Files are stored directly in
// Asterisk's sound directory so ASTERISK_ANNOUNCEMENT can reference them
// by name without any extra copy step.
type AnnouncementServer struct {
	SoundsDir string // e.g. /var/lib/asterisk/sounds/custom
}

func NewAnnouncementServer(soundsDir string) *AnnouncementServer {
	return &AnnouncementServer{SoundsDir: soundsDir}
}

// HandleList returns the names of all uploaded announcements (without the
// .wav extension), suitable for use as ASTERISK_ANNOUNCEMENT.
func (s *AnnouncementServer) HandleList(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(s.SoundsDir)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, map[string][]string{"announcements": {}})
			return
		}
		http.Error(w, "could not read sounds directory", http.StatusInternalServerError)
		return
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".wav" {
			continue
		}
		names = append(names, e.Name()[:len(e.Name())-len(".wav")])
	}
	sort.Strings(names)
	writeJSON(w, map[string][]string{"announcements": names})
}

// HandleUpload accepts a multipart upload (fields "name" and "file") and
// stores it as <name>.wav under SoundsDir. The file must already be in a
// format Asterisk can play (8kHz mono WAV recommended) — no transcoding is
// performed.
func (s *AnnouncementServer) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MiB cap
		http.Error(w, "could not parse upload", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if !safeAnnouncementName.MatchString(name) {
		http.Error(w, "name must match ^[a-zA-Z0-9_-]{1,64}$", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if err := os.MkdirAll(s.SoundsDir, 0o755); err != nil {
		http.Error(w, "could not create sounds directory", http.StatusInternalServerError)
		return
	}

	dest, err := os.Create(filepath.Join(s.SoundsDir, name+".wav"))
	if err != nil {
		http.Error(w, "could not store file", http.StatusInternalServerError)
		return
	}
	defer dest.Close()
	if _, err := io.Copy(dest, file); err != nil {
		http.Error(w, "could not store file", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func randomHex32() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte("fallback-" + string(rune(0))))
	}
	return hex.EncodeToString(b)
}
