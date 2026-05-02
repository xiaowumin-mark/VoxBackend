package audio

import (
	"errors"
	"sync"
)

var ErrRingClosed = errors.New("音频环形缓冲区已关闭")

// Ring 是播放线程和处理线程之间的阻塞环形缓冲区。
type Ring struct {
	mu       sync.Mutex
	notEmpty *sync.Cond
	notFull  *sync.Cond

	buf    []Sample
	head   int
	tail   int
	count  int
	closed bool
	err    error
}

func NewRing(capacity int) *Ring {
	if capacity < 1 {
		capacity = 1
	}
	r := &Ring{buf: make([]Sample, capacity)}
	r.notEmpty = sync.NewCond(&r.mu)
	r.notFull = sync.NewCond(&r.mu)
	return r
}

func (r *Ring) Write(samples []Sample) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	written := 0
	for written < len(samples) {
		for r.count == len(r.buf) && !r.closed {
			r.notFull.Wait()
		}
		if r.closed {
			if r.err != nil {
				return r.err
			}
			return ErrRingClosed
		}

		space := len(r.buf) - r.count
		if space == 0 {
			continue
		}
		n := minInt(len(samples)-written, space)
		first := minInt(n, len(r.buf)-r.tail)
		copy(r.buf[r.tail:r.tail+first], samples[written:written+first])
		r.tail = (r.tail + first) % len(r.buf)
		r.count += first
		written += first

		second := n - first
		if second > 0 {
			copy(r.buf[r.tail:r.tail+second], samples[written:written+second])
			r.tail = (r.tail + second) % len(r.buf)
			r.count += second
			written += second
		}
		r.notEmpty.Signal()
	}
	return nil
}

func (r *Ring) MixForPlayback(dst []Sample, mixer *Mixer) (int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	want := 2 * len(dst)
	for r.count < want && !r.closed {
		r.notEmpty.Wait()
	}
	if r.count == 0 && r.closed {
		return 0, false
	}

	n := minInt(len(dst), r.count/2)
	for i := 0; i < n; i++ {
		vocals := r.buf[r.head]
		r.head = (r.head + 1) % len(r.buf)
		r.count--

		accomp := r.buf[r.head]
		r.head = (r.head + 1) % len(r.buf)
		r.count--

		dst[i] = mixer.MixSingle(vocals, accomp)
	}
	r.notFull.Signal()
	return n, true
}

func (r *Ring) ReadPairs(vocal, accomp []Sample) (int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	want := 2 * minInt(len(vocal), len(accomp))
	for r.count < want && !r.closed {
		r.notEmpty.Wait()
	}
	if r.count == 0 && r.closed {
		return 0, false
	}

	chunk := r.count / 2
	n := minInt(len(vocal), len(accomp))
	if chunk < n {
		n = chunk
	}
	for i := 0; i < n; i++ {
		vocal[i] = r.buf[r.head]
		r.head = (r.head + 1) % len(r.buf)
		r.count--
		accomp[i] = r.buf[r.head]
		r.head = (r.head + 1) % len(r.buf)
		r.count--
	}
	r.notFull.Signal()
	return n, true
}

func (r *Ring) DiscardAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.head = 0
	r.tail = 0
	r.count = 0
	r.notFull.Broadcast()
}

// DiscardAndReopen 清空数据并恢复写入状态，主要用于 seek。
func (r *Ring) DiscardAndReopen() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.head = 0
	r.tail = 0
	r.count = 0
	r.closed = false
	r.err = nil
	r.notFull.Broadcast()
	r.notEmpty.Broadcast()
}

func (r *Ring) CloseWithError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	r.closed = true
	r.err = err
	r.notEmpty.Broadcast()
	r.notFull.Broadcast()
}

func (r *Ring) Err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

func (r *Ring) Closed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closed
}

func (r *Ring) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.count
}

func (r *Ring) WaitForAtLeast(target int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for r.count < target && !r.closed {
		r.notEmpty.Wait()
	}
	if r.count >= target {
		return nil
	}
	if r.err != nil {
		return r.err
	}
	return ErrRingClosed
}
