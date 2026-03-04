package recorder

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
)

// Recorder — запись аудио через arecord (ALSA).
type Recorder struct {
	device     string
	sampleRate int

	mu     sync.Mutex
	cmd    *exec.Cmd
	cancel context.CancelFunc
}

// New создаёт рекордер с указанным ALSA-устройством и частотой дискретизации.
func New(device string, sampleRate int) *Recorder {
	return &Recorder{
		device:     device,
		sampleRate: sampleRate,
	}
}

// Start запускает непрерывную запись и возвращает поток PCM-данных (S16_LE, mono).
func (r *Recorder) Start(ctx context.Context) (io.ReadCloser, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cmd != nil {
		return nil, fmt.Errorf("recorder already running")
	}

	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	r.cmd = exec.CommandContext(ctx, "arecord",
		"-D", r.device,
		"-f", "S16_LE",
		"-r", fmt.Sprintf("%d", r.sampleRate),
		"-c", "1",
		"-t", "raw",
	)

	stdout, err := r.cmd.StdoutPipe()
	if err != nil {
		cancel()
		r.cmd = nil
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := r.cmd.Start(); err != nil {
		cancel()
		r.cmd = nil
		return nil, fmt.Errorf("start arecord: %w", err)
	}

	log.Printf("[recorder] запись запущена: device=%s rate=%d", r.device, r.sampleRate)

	go func() {
		if err := r.cmd.Wait(); err != nil && ctx.Err() == nil {
			log.Printf("[recorder] arecord завершился: %v", err)
		}
		r.mu.Lock()
		r.cmd = nil
		r.mu.Unlock()
	}()

	return stdout, nil
}

// Stop останавливает запись.
func (r *Recorder) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
		log.Println("[recorder] запись остановлена")
	}
}
