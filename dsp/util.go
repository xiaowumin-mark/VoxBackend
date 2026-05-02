package dsp

import "github.com/xiaowumin-mark/VoxBackend/audio"

func deinterleavePairs(raw []audio.Sample, vocal, accomp []audio.Sample) int {
	n := minInt(len(vocal), len(accomp))
	if 2*n > len(raw) {
		n = len(raw) / 2
	}
	for i := 0; i < n; i++ {
		vocal[i] = raw[2*i]
		accomp[i] = raw[2*i+1]
	}
	return n
}

func interleavePairs(dst, vocal, accomp []audio.Sample) int {
	n := minInt(len(vocal), len(accomp))
	if 2*n > len(dst) {
		n = len(dst) / 2
	}
	for i := 0; i < n; i++ {
		dst[2*i] = vocal[i]
		dst[2*i+1] = accomp[i]
	}
	return n
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
