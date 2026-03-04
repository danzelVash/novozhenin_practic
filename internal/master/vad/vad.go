package vad

import (
	"encoding/binary"
	"io"
	"log"
	"math"
	"time"
)

// VAD — детектор голосовой активности на основе RMS.
type VAD struct {
	threshold  float64
	silenceDur time.Duration
	sampleRate int
}

// New создаёт VAD с порогом RMS, длительностью тишины и частотой дискретизации.
func New(threshold float64, silenceDur time.Duration, sampleRate int) *VAD {
	return &VAD{
		threshold:  threshold,
		silenceDur: silenceDur,
		sampleRate: sampleRate,
	}
}

// Process читает PCM-поток и отправляет обнаруженные фразы в канал.
func (v *VAD) Process(reader io.Reader, phrases chan<- []byte) {
	chunkSamples := v.sampleRate / 50
	chunkBytes := chunkSamples * 2
	buf := make([]byte, chunkBytes)

	var (
		speechBuf    []byte
		isSpeech     bool
		silenceStart time.Time
	)

	for {
		n, err := io.ReadFull(reader, buf)
		if n > 0 {
			rms := calcRMS(buf[:n])

			if rms >= v.threshold {
				if !isSpeech {
					isSpeech = true
					log.Printf("[vad] речь началась (rms=%.4f)", rms)
				}
				silenceStart = time.Time{}
				speechBuf = append(speechBuf, buf[:n]...)
			} else if isSpeech {
				speechBuf = append(speechBuf, buf[:n]...)

				if silenceStart.IsZero() {
					silenceStart = time.Now()
				} else if time.Since(silenceStart) >= v.silenceDur {
					log.Printf("[vad] речь завершена, %d байт (%.1fс)",
						len(speechBuf),
						float64(len(speechBuf))/float64(v.sampleRate*2))

					phrase := make([]byte, len(speechBuf))
					copy(phrase, speechBuf)
					phrases <- phrase

					speechBuf = speechBuf[:0]
					isSpeech = false
					silenceStart = time.Time{}
				}
			}
		}

		if err != nil {
			if err != io.EOF && err != io.ErrUnexpectedEOF {
				log.Printf("[vad] ошибка чтения: %v", err)
			}
			break
		}
	}

	if len(speechBuf) > 0 {
		phrases <- speechBuf
	}
}

// calcRMS вычисляет среднеквадратичное значение амплитуды PCM-данных.
func calcRMS(data []byte) float64 {
	samples := len(data) / 2
	if samples == 0 {
		return 0
	}

	var sumSq float64
	for i := 0; i < samples; i++ {
		sample := int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2]))
		normalized := float64(sample) / 32768.0
		sumSq += normalized * normalized
	}

	return math.Sqrt(sumSq / float64(samples))
}
