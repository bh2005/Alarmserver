package espeak

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSynthesize_RunsEspeakThenSox(t *testing.T) {
	dir := t.TempDir()
	var calls [][]string

	s := New(Config{SoundsDir: dir, Voice: "de"})
	s.runCmd = func(ctx context.Context, name string, args ...string) error {
		calls = append(calls, append([]string{name}, args...))
		return nil
	}

	soundName, cleanup, err := s.Synthesize(context.Background(), "Feueralarm in Halle 3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	if !strings.HasPrefix(soundName, filepath.Base(dir)+"/tts-") {
		t.Fatalf("unexpected sound name: %q", soundName)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 external calls (espeak-ng, sox), got %d: %v", len(calls), calls)
	}
	if calls[0][0] != "espeak-ng" {
		t.Fatalf("expected first call to be espeak-ng, got %q", calls[0][0])
	}
	if calls[1][0] != "sox" {
		t.Fatalf("expected second call to be sox, got %q", calls[1][0])
	}
}

func TestSynthesize_CleanupRemovesFinalFile(t *testing.T) {
	dir := t.TempDir()
	s := New(Config{SoundsDir: dir})
	s.runCmd = func(ctx context.Context, name string, args ...string) error {
		// Simulate the real binaries by creating whatever output file
		// they were asked to write (last positional arg after -w / before
		// resample target).
		if name == "espeak-ng" {
			// args: -v <voice> -w <rawPath> [-s <speed>] <text>
			return os.WriteFile(args[3], []byte("raw"), 0o644)
		}
		// sox: <raw> -r 8000 -c 1 <final>
		return os.WriteFile(args[len(args)-1], []byte("final"), 0o644)
	}

	soundName, cleanup, err := s.Synthesize(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	finalPath := filepath.Join(dir, filepath.Base(soundName)+".wav")
	if _, err := os.Stat(finalPath); err != nil {
		t.Fatalf("expected final file to exist: %v", err)
	}

	cleanup()
	if _, err := os.Stat(finalPath); !os.IsNotExist(err) {
		t.Fatalf("expected final file to be removed after cleanup, stat err: %v", err)
	}
}

func TestSynthesize_EspeakFailureCleansUpAndReturnsError(t *testing.T) {
	dir := t.TempDir()
	s := New(Config{SoundsDir: dir})
	s.runCmd = func(ctx context.Context, name string, args ...string) error {
		if name == "espeak-ng" {
			return errors.New("voice not found")
		}
		t.Fatal("sox should not be called when espeak-ng fails")
		return nil
	}

	_, _, err := s.Synthesize(context.Background(), "test")
	if err == nil || !strings.Contains(err.Error(), "voice not found") {
		t.Fatalf("expected wrapped espeak-ng error, got: %v", err)
	}
}

func TestSynthesize_SoxFailureCleansUpAndReturnsError(t *testing.T) {
	dir := t.TempDir()
	s := New(Config{SoundsDir: dir})
	s.runCmd = func(ctx context.Context, name string, args ...string) error {
		if name == "espeak-ng" {
			return nil
		}
		return errors.New("unsupported format")
	}

	_, _, err := s.Synthesize(context.Background(), "test")
	if err == nil || !strings.Contains(err.Error(), "unsupported format") {
		t.Fatalf("expected wrapped sox error, got: %v", err)
	}
}
