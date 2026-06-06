package player

import (
	"context"
	"testing"
)

func TestPlaylistManagerRestoresSourceAtTail(t *testing.T) {
	tracks := []Track{
		{Path: "a.wav", Title: "A"},
		{Path: "b.wav", Title: "B"},
	}
	pm := NewPlaylistManager(context.Background(), DefaultConfig(), tracks, defaultSampleRate, func(Event) {})
	pm.currentIndex = 1
	pm.current = &PipelineBundle{index: 1, track: cloneTrack(tracks[1])}

	pm.ensureSourceRestoredForTailLocked()

	if got, want := len(pm.tracks), 4; got != want {
		t.Fatalf("track count after restore = %d, want %d", got, want)
	}
	if got, want := pm.tracks[2].Path, "a.wav"; got != want {
		t.Fatalf("restored first path = %q, want %q", got, want)
	}
	if got, want := len(pm.sourceTracks), 2; got != want {
		t.Fatalf("source track count = %d, want %d", got, want)
	}
}

func TestPlaylistManagerCompactsRestoredSource(t *testing.T) {
	tracks := []Track{
		{Path: "a.wav", Title: "A"},
		{Path: "b.wav", Title: "B"},
	}
	pm := NewPlaylistManager(context.Background(), DefaultConfig(), tracks, defaultSampleRate, func(Event) {})
	pm.tracks = append(pm.tracks, cloneTracks(tracks)...)
	pm.currentIndex = 2
	pm.current = &PipelineBundle{index: 2, track: cloneTrack(tracks[0])}

	pm.compactRestoredSourceLocked()

	if got, want := len(pm.tracks), 2; got != want {
		t.Fatalf("track count after compact = %d, want %d", got, want)
	}
	if got, want := pm.currentIndex, 0; got != want {
		t.Fatalf("current index after compact = %d, want %d", got, want)
	}
	if got, want := pm.current.index, 0; got != want {
		t.Fatalf("bundle index after compact = %d, want %d", got, want)
	}
}

func TestPlaylistManagerMoveToTailUpdatesSourceOrder(t *testing.T) {
	tracks := []Track{
		{Path: "a.wav", Title: "A"},
		{Path: "b.wav", Title: "B"},
		{Path: "c.wav", Title: "C"},
	}
	pm := NewPlaylistManager(context.Background(), DefaultConfig(), tracks, defaultSampleRate, func(Event) {})

	pm.handleMoveTrackLocked(1, len(tracks))

	want := []string{"a.wav", "c.wav", "b.wav"}
	for i, path := range want {
		if got := pm.tracks[i].Path; got != path {
			t.Fatalf("tracks[%d] = %q, want %q", i, got, path)
		}
		if got := pm.sourceTracks[i].Path; got != path {
			t.Fatalf("sourceTracks[%d] = %q, want %q", i, got, path)
		}
	}
}
