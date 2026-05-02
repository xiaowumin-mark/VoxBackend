package separator

import "github.com/xiaowumin-mark/VoxBackend/audio"

func computeResidualMask(magnitudes, vocalMasked []float32) []float32 {
	out := make([]float32, len(magnitudes))
	for i := range magnitudes {
		out[i] = magnitudes[i] - vocalMasked[i]
		if out[i] < 0 {
			out[i] = 0
		}
	}
	return out
}

func cloneSamples(src []audio.Sample) []audio.Sample {
	dst := make([]audio.Sample, len(src))
	copy(dst, src)
	return dst
}

func subtractSamples(a, b []audio.Sample) []audio.Sample {
	n := minInt(len(a), len(b))
	out := make([]audio.Sample, n)
	for i := 0; i < n; i++ {
		out[i][0] = a[i][0] - b[i][0]
		out[i][1] = a[i][1] - b[i][1]
	}
	return out
}

func shiftSamplesLeft(src []audio.Sample, n int) []audio.Sample {
	if n <= 0 {
		return src
	}
	if n >= len(src) {
		return nil
	}
	copy(src, src[n:])
	tail := src[len(src)-n:]
	for i := range tail {
		tail[i] = audio.Sample{}
	}
	return src[:len(src)-n]
}

func shiftFloat64Left(src []float64, n int) []float64 {
	if n <= 0 {
		return src
	}
	if n >= len(src) {
		return nil
	}
	copy(src, src[n:])
	tail := src[len(src)-n:]
	for i := range tail {
		tail[i] = 0
	}
	return src[:len(src)-n]
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
