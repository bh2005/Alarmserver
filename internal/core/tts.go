package core

import "context"

// Synthesizer renders text to a temporary announcement that a voice
// notifier can play back. soundName is the value to pass as an Asterisk
// Playback Data argument (relative to its sound search path, without a
// file extension, e.g. "custom/3f9a2c"). cleanup removes the generated
// file once it is no longer needed and must always be called.
type Synthesizer interface {
	Synthesize(ctx context.Context, text string) (soundName string, cleanup func(), err error)
}
