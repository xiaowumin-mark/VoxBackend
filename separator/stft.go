package separator

import (
	"fmt"
	"math"
	"math/cmplx"

	"github.com/xiaowumin-mark/VoxBackend/audio"
	"gonum.org/v1/gonum/dsp/fourier"
)

type STFTProcessor struct {
	fft     *fourier.FFT
	fftSize int
	hopSize int
	window  []float64
}

func NewSTFTProcessor(fftSize, hopSize int) *STFTProcessor {
	window := make([]float64, fftSize)
	for i := range window {
		window[i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(fftSize))
	}
	return &STFTProcessor{
		fft:     fourier.NewFFT(fftSize),
		fftSize: fftSize,
		hopSize: hopSize,
		window:  window,
	}
}

func (p *STFTProcessor) WindowSamples(frames int) int {
	return p.fftSize + p.hopSize*(frames-1)
}

func (p *STFTProcessor) EncodeStereo(segment []audio.Sample, frames int) ([][][]complex128, []float32, error) {
	if len(segment) != p.WindowSamples(frames) {
		return nil, nil, fmt.Errorf(
			"STFT 期望 %d 个采样点，实际得到 %d",
			p.WindowSamples(frames),
			len(segment),
		)
	}

	bins := p.fftSize/2 + 1
	coeffs := make([][][]complex128, 2)
	magnitudes := make([]float32, 2*bins*frames)
	timeBuf := make([]float64, p.fftSize)
	freqBuf := make([]complex128, bins)

	for ch := 0; ch < 2; ch++ {
		coeffs[ch] = make([][]complex128, frames)
		for frame := 0; frame < frames; frame++ {
			start := frame * p.hopSize
			for i := 0; i < p.fftSize; i++ {
				timeBuf[i] = segment[start+i][ch] * p.window[i]
			}

			spec := p.fft.Coefficients(freqBuf[:bins], timeBuf)
			frameCoeffs := make([]complex128, bins)
			copy(frameCoeffs, spec)
			coeffs[ch][frame] = frameCoeffs

			for bin := 0; bin < bins; bin++ {
				magnitudes[specIndex(ch, bin, frame, bins, frames)] = float32(cmplx.Abs(frameCoeffs[bin]))
			}
		}
	}

	return coeffs, magnitudes, nil
}

// EncodeStereoComplex4 将双声道时域信号编码为 4 通道复频谱：
// [左实部, 左虚部, 右实部, 右虚部]。
func (p *STFTProcessor) EncodeStereoComplex4(segment []audio.Sample, frames, freqBins int) ([]float32, error) {
	if len(segment) != p.WindowSamples(frames) {
		return nil, fmt.Errorf(
			"STFT 期望 %d 个采样点，实际得到 %d",
			p.WindowSamples(frames),
			len(segment),
		)
	}

	fullBins := p.fftSize/2 + 1
	if freqBins <= 0 || freqBins > fullBins {
		return nil, fmt.Errorf("频率 bin 数无效：%d，期望范围是 [1, %d]", freqBins, fullBins)
	}

	features := make([]float32, 4*freqBins*frames)
	timeBuf := make([]float64, p.fftSize)
	freqBuf := make([]complex128, fullBins)

	for ch := 0; ch < 2; ch++ {
		for frame := 0; frame < frames; frame++ {
			start := frame * p.hopSize
			for i := 0; i < p.fftSize; i++ {
				timeBuf[i] = segment[start+i][ch] * p.window[i]
			}

			spec := p.fft.Coefficients(freqBuf[:fullBins], timeBuf)
			for bin := 0; bin < freqBins; bin++ {
				c := spec[bin]
				features[tensorIndex(ch*2, bin, frame, 4, freqBins, frames)] = float32(real(c))
				features[tensorIndex(ch*2+1, bin, frame, 4, freqBins, frames)] = float32(imag(c))
			}
		}
	}
	return features, nil
}

func (p *STFTProcessor) DecodeStereo(masked []float32, coeffs [][][]complex128, frames int) ([]audio.Sample, []float64, error) {
	bins := p.fftSize/2 + 1
	expected := 2 * bins * frames
	if len(masked) != expected {
		return nil, nil, fmt.Errorf("ISTFT 期望 %d 个掩蔽频谱 bin，实际得到 %d", expected, len(masked))
	}
	if len(coeffs) != 2 {
		return nil, nil, fmt.Errorf("ISTFT 期望 2 个声道，实际得到 %d", len(coeffs))
	}

	outLen := p.WindowSamples(frames)
	output := make([]audio.Sample, outLen)
	norm := make([]float64, outLen)
	timeBuf := make([]float64, p.fftSize)

	for ch := 0; ch < 2; ch++ {
		if len(coeffs[ch]) != frames {
			return nil, nil, fmt.Errorf("ISTFT 在声道 %d 上期望 %d 帧，实际得到 %d", ch, frames, len(coeffs[ch]))
		}
		for frame := 0; frame < frames; frame++ {
			if len(coeffs[ch][frame]) != bins {
				return nil, nil, fmt.Errorf(
					"ISTFT 在声道 %d 第 %d 帧期望 %d 个 bin，实际得到 %d",
					ch,
					frame,
					bins,
					len(coeffs[ch][frame]),
				)
			}

			rebuilt := make([]complex128, bins)
			for bin := 0; bin < bins; bin++ {
				original := coeffs[ch][frame][bin]
				magnitude := cmplx.Abs(original)
				if magnitude == 0 {
					rebuilt[bin] = 0
					continue
				}
				maskValue := float64(masked[specIndex(ch, bin, frame, bins, frames)])
				rebuilt[bin] = original / complex(magnitude, 0) * complex(maskValue, 0)
			}

			timeSeq := p.fft.Sequence(timeBuf[:p.fftSize], rebuilt)
			start := frame * p.hopSize
			for i := 0; i < p.fftSize; i++ {
				sample := (timeSeq[i] / float64(p.fftSize)) * p.window[i]
				output[start+i][ch] += sample
				if ch == 0 {
					norm[start+i] += p.window[i] * p.window[i]
				}
			}
		}
	}

	return output, norm, nil
}

// DecodeStereoComplex4 将 4 通道复频谱反变换为双声道时域信号。
func (p *STFTProcessor) DecodeStereoComplex4(features []float32, frames, freqBins int) ([]audio.Sample, []float64, error) {
	fullBins := p.fftSize/2 + 1
	if freqBins <= 0 || freqBins > fullBins {
		return nil, nil, fmt.Errorf("频率 bin 数无效：%d，期望范围是 [1, %d]", freqBins, fullBins)
	}

	expected := 4 * freqBins * frames
	if len(features) != expected {
		return nil, nil, fmt.Errorf("ISTFT 期望 %d 个复频谱特征，实际得到 %d", expected, len(features))
	}

	outLen := p.WindowSamples(frames)
	output := make([]audio.Sample, outLen)
	norm := make([]float64, outLen)
	timeBuf := make([]float64, p.fftSize)

	for ch := 0; ch < 2; ch++ {
		for frame := 0; frame < frames; frame++ {
			rebuilt := make([]complex128, fullBins)
			for bin := 0; bin < freqBins; bin++ {
				re := features[tensorIndex(ch*2, bin, frame, 4, freqBins, frames)]
				im := features[tensorIndex(ch*2+1, bin, frame, 4, freqBins, frames)]
				rebuilt[bin] = complex(float64(re), float64(im))
			}

			timeSeq := p.fft.Sequence(timeBuf[:p.fftSize], rebuilt)
			start := frame * p.hopSize
			for i := 0; i < p.fftSize; i++ {
				sample := (timeSeq[i] / float64(p.fftSize)) * p.window[i]
				output[start+i][ch] += sample
				if ch == 0 {
					norm[start+i] += p.window[i] * p.window[i]
				}
			}
		}
	}

	return output, norm, nil
}

func specIndex(channel, bin, frame, bins, frames int) int {
	return tensorIndex(channel, bin, frame, 2, bins, frames)
}

func tensorIndex(channel, bin, frame, channels, bins, frames int) int {
	_ = channels
	return (channel*bins+bin)*frames + frame
}
