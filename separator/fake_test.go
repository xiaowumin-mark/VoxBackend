package separator

import (
	"testing"

	"github.com/xiaowumin-mark/VoxBackend/audio"
)

func TestFakeSeparatorReconstructsMixAtUnityGain(t *testing.T) {
	sep := NewFake(0.8, 1.0, 0)
	input := []audio.Sample{
		{0.6, 0.2},
		{-0.4, 0.1},
		{0.0, -0.5},
	}

	var out Chunk
	if err := sep.Process(&out, input); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	for i := range input {
		gotL := out.Vocals[i][0] + out.Accomp[i][0]
		gotR := out.Vocals[i][1] + out.Accomp[i][1]
		if gotL != input[i][0] || gotR != input[i][1] {
			t.Fatalf("sample %d reconstructed to [%f %f], want [%f %f]", i, gotL, gotR, input[i][0], input[i][1])
		}
	}
}

func TestFakeSeparatorDrainFlushesLatencyTail(t *testing.T) {
	sep := NewFake(0.8, 1.0, 2)
	input := []audio.Sample{
		{0.1, 0.2},
		{0.3, 0.4},
		{0.5, 0.6},
	}

	var out Chunk
	if err := sep.Process(&out, input); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if got := len(sep.pending); got != 2 {
		t.Fatalf("pending = %d, want 2", got)
	}

	n, err := sep.Drain(&out, 8)
	if err != nil {
		t.Fatalf("Drain() error = %v", err)
	}
	if n != 2 {
		t.Fatalf("Drain() emitted %d, want 2", n)
	}
	if len(sep.pending) != 0 {
		t.Fatalf("pending after drain = %d, want 0", len(sep.pending))
	}
}
