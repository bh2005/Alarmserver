// Package espeak implements core.Synthesizer using the offline espeak-ng
// TTS engine plus sox to resample to Asterisk's expected 8kHz mono WAV.
// Everything runs locally on the host — no alert text is ever sent to a
// third-party cloud TTS API, which matters for sensitive alert content.
//
// Requires the espeak-ng and sox packages (see deploy/install-asterisk.sh).
package espeak

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Config struct {
	SoundsDir string // Asterisk sound dir to write into, e.g. /var/lib/asterisk/sounds/custom
	Voice     string // espeak-ng voice, e.g. "de" or "de+f3"; default "de"
	Speed     int    // words per minute (espeak-ng -s); 0 = engine default
}

type Synthesizer struct {
	cfg Config
	// runCmd is overridable in tests to avoid invoking real binaries.
	runCmd func(ctx context.Context, name string, args ...string) error
}

func New(cfg Config) *Synthesizer {
	if cfg.Voice == "" {
		cfg.Voice = "de"
	}
	return &Synthesizer{cfg: cfg, runCmd: runExternal}
}

func runExternal(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w: %s", name, err, string(out))
	}
	return nil
}

// Synthesize renders text to a WAV file under SoundsDir and returns the
// Asterisk-relative sound name (directory basename + filename, no
// extension) plus a cleanup func that removes the generated file(s).
func (s *Synthesizer) Synthesize(ctx context.Context, text string) (string, func(), error) {
	id, err := randomID()
	if err != nil {
		return "", nil, fmt.Errorf("espeak: generate id: %w", err)
	}

	if err := os.MkdirAll(s.cfg.SoundsDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("espeak: sounds dir: %w", err)
	}

	rawPath := filepath.Join(s.cfg.SoundsDir, "tts-"+id+"-raw.wav")
	finalPath := filepath.Join(s.cfg.SoundsDir, "tts-"+id+".wav")
	cleanup := func() {
		os.Remove(rawPath)
		os.Remove(finalPath)
	}

	args := []string{"-v", s.cfg.Voice, "-w", rawPath}
	if s.cfg.Speed > 0 {
		args = append(args, "-s", fmt.Sprintf("%d", s.cfg.Speed))
	}
	args = append(args, text)
	if err := s.runCmd(ctx, "espeak-ng", args...); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("espeak: synthesize: %w", err)
	}

	// Asterisk's Playback expects 8kHz mono; resample what espeak-ng produced.
	if err := s.runCmd(ctx, "sox", rawPath, "-r", "8000", "-c", "1", finalPath); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("espeak: resample: %w", err)
	}
	os.Remove(rawPath)

	soundName := filepath.Base(s.cfg.SoundsDir) + "/tts-" + id
	return soundName, cleanup, nil
}

func randomID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
