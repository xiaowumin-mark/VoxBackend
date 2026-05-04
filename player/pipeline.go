package player

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/gopxl/beep/v2"
	"github.com/xiaowumin-mark/VoxBackend/audio"
	"github.com/xiaowumin-mark/VoxBackend/separator"
)

var ErrPipelineStopped = errors.New("音频处理流水线已停止")

type seekRequest struct {
	sampleOffset int64
	result       chan error
}

type Pipeline struct {
	srcCloser io.Closer
	sep       separator.Separator
	ring      *audio.Ring

	rawSeeker     beep.StreamSeeker
	rawSampleRate beep.SampleRate
	outSampleRate beep.SampleRate
	needResample  bool

	seekCh     chan seekRequest
	done       chan struct{}
	stop       sync.Once
	onSeekDone func()
}

func NewPipeline(
	rawStream beep.StreamSeeker,
	srcCloser io.Closer,
	sep separator.Separator,
	ring *audio.Ring,
	rawSR beep.SampleRate,
	outSR beep.SampleRate,
) *Pipeline {
	return &Pipeline{
		srcCloser:     srcCloser,
		sep:           sep,
		ring:          ring,
		rawSeeker:     rawStream,
		rawSampleRate: rawSR,
		outSampleRate: outSR,
		needResample:  rawSR != outSR,
		seekCh:        make(chan seekRequest, 1),
		done:          make(chan struct{}),
	}
}

func (p *Pipeline) buildStream() beep.Streamer {
	if p.needResample {
		return beep.Resample(4, p.rawSampleRate, p.outSampleRate, p.rawSeeker)
	}
	return p.rawSeeker
}

func (p *Pipeline) Run() {
	defer close(p.done)
	if p.srcCloser != nil {
		defer p.srcCloser.Close()
	}
	defer p.sep.Close()

	input := make([]audio.Sample, ChunkSize)
	var stems separator.Chunk
	output := make([]audio.Sample, 2*ChunkSize)

	streamer := p.buildStream()

	for {
		select {
		case req := <-p.seekCh:
			err := p.handleSeek(req.sampleOffset, &streamer, input, output, &stems)
			if req.result != nil {
				req.result <- err
			}
			continue
		default:
		}

		n, ok := streamer.Stream(input)
		if n > 0 {
			chunk := input[:n]
			if err := p.sep.Process(&stems, chunk); err != nil {
				p.ring.CloseWithError(fmt.Errorf("分离器处理失败: %w", err))
				return
			}
			select {
			case req := <-p.seekCh:
				err := p.handleSeek(req.sampleOffset, &streamer, input, output, &stems)
				if req.result != nil {
					req.result <- err
				}
				continue
			default:
			}
			if len(stems.Vocals) < n || len(stems.Accomp) < n {
				p.ring.CloseWithError(errors.New("分离器返回了长度不足的音频块"))
				return
			}
			for i := 0; i < n; i++ {
				output[2*i] = stems.Vocals[i]
				output[2*i+1] = stems.Accomp[i]
			}
			if err := p.ring.Write(output[:2*n]); err != nil {
				p.ring.CloseWithError(err)
				return
			}
		}

		if !ok {
			select {
			case req := <-p.seekCh:
				err := p.handleSeek(req.sampleOffset, &streamer, input, output, &stems)
				if req.result != nil {
					req.result <- err
				}
				continue
			default:
			}

			if err := streamer.Err(); err != nil {
				p.ring.CloseWithError(fmt.Errorf("输入音频流错误: %w", err))
				return
			}
			if drainer, ok := p.sep.(separator.Drainable); ok {
				for {
					select {
					case req := <-p.seekCh:
						err := p.handleSeek(req.sampleOffset, &streamer, input, output, &stems)
						if req.result != nil {
							req.result <- err
						}
						goto continueMain
					default:
					}
					n, err := drainer.Drain(&stems, ChunkSize)
					if err != nil {
						p.ring.CloseWithError(fmt.Errorf("分离器排空失败: %w", err))
						return
					}
					if n == 0 {
						break
					}
					for i := 0; i < n; i++ {
						output[2*i] = stems.Vocals[i]
						output[2*i+1] = stems.Accomp[i]
					}
					if err := p.ring.Write(output[:2*n]); err != nil {
						p.ring.CloseWithError(err)
						return
					}
				}
			}
			p.ring.CloseWithError(nil)
			return
		}
	continueMain:
	}
}

func (p *Pipeline) handleSeek(
	sampleOffset int64,
	streamer *beep.Streamer,
	input []audio.Sample,
	output []audio.Sample,
	stems *separator.Chunk,
) error {
	rawOffset := sampleOffset
	if p.needResample {
		rawOffset = int64(float64(sampleOffset) * float64(p.rawSampleRate) / float64(p.outSampleRate))
	}
	if err := p.rawSeeker.Seek(int(rawOffset)); err != nil {
		return fmt.Errorf("seek 源音频失败: %w", err)
	}

	*streamer = p.buildStream()
	p.sep.Reset()
	p.ring.DiscardAndReopen()

	for {
		select {
		case <-p.sep.WarmupReady():
			goto phase2
		default:
		}
		n, ok := (*streamer).Stream(input)
		if n > 0 {
			if err := p.sep.Process(stems, input[:n]); err != nil {
				return fmt.Errorf("seek 预热失败: %w", err)
			}
		}
		if !ok {
			break
		}
	}

phase2:
	p.sep.ResetOutput()
	if err := p.rawSeeker.Seek(int(rawOffset)); err != nil {
		return fmt.Errorf("seek 源音频(阶段2)失败: %w", err)
	}
	*streamer = p.buildStream()
	p.ring.DiscardAndReopen()

	prefill := PrefillTargetSamples(p.sep)
	written := 0
	waitingForRealOutput := true
	for written < prefill {
		n, ok := (*streamer).Stream(input)
		if n > 0 {
			chunk := input[:n]
			if err := p.sep.Process(stems, chunk); err != nil {
				return fmt.Errorf("seek 预填充失败: %w", err)
			}
			if len(stems.Vocals) < n || len(stems.Accomp) < n {
				return errors.New("seek 预填充：分离器返回长度不足")
			}
			if waitingForRealOutput {
				if reporter, hasReporter := p.sep.(interface{ LastRealOutputSamples() int }); hasReporter {
					if reporter.LastRealOutputSamples() == 0 {
						continue
					}
				}
				waitingForRealOutput = false
				p.ring.DiscardAndReopen()
			}
			for i := 0; i < n; i++ {
				output[2*i] = stems.Vocals[i]
				output[2*i+1] = stems.Accomp[i]
			}
			if err := p.ring.Write(output[:2*n]); err != nil {
				return err
			}
			written += n
		}
		if !ok {
			break
		}
	}
	if p.onSeekDone != nil {
		p.onSeekDone()
	}
	return nil
}

func (p *Pipeline) SeekSamples(sampleOffset int64) error {
	result := make(chan error, 1)
	req := seekRequest{sampleOffset: sampleOffset, result: result}
	select {
	case p.seekCh <- req:
	case <-p.done:
		return ErrPipelineStopped
	}
	select {
	case err := <-result:
		return err
	case <-p.done:
		return ErrPipelineStopped
	}
}

func (p *Pipeline) SeekSamplesAsync(sampleOffset int64) {
	req := seekRequest{sampleOffset: sampleOffset, result: nil}
	select {
	case p.seekCh <- req:
	case <-p.done:
	}
}

func (p *Pipeline) SetSeekDoneCallback(fn func()) {
	p.onSeekDone = fn
}

func (p *Pipeline) Stop() {
	p.stop.Do(func() {
		p.ring.CloseWithError(ErrPipelineStopped)
		if p.srcCloser != nil {
			_ = p.srcCloser.Close()
		}
	})
}

func PrefillTargetSamples(sep separator.Separator) int {
	type prefillTargeter interface {
		PrefillTargetSamples() int
	}
	if p, ok := sep.(prefillTargeter); ok {
		return p.PrefillTargetSamples()
	}
	return sep.LatencySamples() + 2*ChunkSize
}
