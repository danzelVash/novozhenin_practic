package bench

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/novozhenin/practic/bench/harness"
	"github.com/novozhenin/practic/internal/transport"
	"github.com/novozhenin/practic/internal/transport/grpctransport"
	"github.com/novozhenin/practic/internal/transport/wstransport"
	pb "github.com/novozhenin/practic/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func init() {
	log.SetOutput(io.Discard)
}

// ---------------------------------------------------------------------------
// A. Throughput (end-to-end msgs/sec)
// ---------------------------------------------------------------------------

func benchThroughput(b *testing.B, h *harness.TransportHarness) {
	b.Helper()
	defer h.Teardown()

	var received int64
	h.Handler = func(_ transport.Command) {
		atomic.AddInt64(&received, 1)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		h.Pub.Publish(transport.Command{DirectionUp: i%2 == 0})
	}

	// Wait for all messages to be received.
	deadline := time.After(30 * time.Second)
	for atomic.LoadInt64(&received) < int64(b.N) {
		select {
		case <-deadline:
			b.Fatalf("throughput: received %d/%d messages", atomic.LoadInt64(&received), b.N)
		default:
			time.Sleep(time.Millisecond)
		}
	}

	b.StopTimer()
}

func BenchmarkThroughput_GRPC(b *testing.B) {
	benchThroughput(b, harness.NewGRPC(b))
}

func BenchmarkThroughput_WS(b *testing.B) {
	benchThroughput(b, harness.NewWS(b))
}

func BenchmarkThroughput_MQTT(b *testing.B) {
	benchThroughput(b, harness.NewMQTT(b))
}

// ---------------------------------------------------------------------------
// B. Latency (per-message, with percentiles)
// ---------------------------------------------------------------------------

func benchLatency(b *testing.B, h *harness.TransportHarness) {
	b.Helper()
	defer h.Teardown()

	ch := make(chan time.Time, 1)
	h.Handler = func(_ transport.Command) {
		ch <- time.Now()
	}

	durations := make([]time.Duration, 0, b.N)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()
		h.Pub.Publish(transport.Command{DirectionUp: i%2 == 0})

		select {
		case end := <-ch:
			durations = append(durations, end.Sub(start))
		case <-time.After(10 * time.Second):
			b.Fatal("latency: timeout waiting for message")
		}
	}

	b.StopTimer()

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	n := len(durations)
	if n == 0 {
		return
	}

	b.ReportMetric(float64(durations[n/2].Microseconds()), "p50-µs")
	b.ReportMetric(float64(durations[n*95/100].Microseconds()), "p95-µs")
	b.ReportMetric(float64(durations[n*99/100].Microseconds()), "p99-µs")
	b.ReportMetric(float64(durations[n-1].Microseconds()), "max-µs")
}

func BenchmarkLatency_GRPC(b *testing.B) {
	benchLatency(b, harness.NewGRPC(b))
}

func BenchmarkLatency_WS(b *testing.B) {
	benchLatency(b, harness.NewWS(b))
}

func BenchmarkLatency_MQTT(b *testing.B) {
	benchLatency(b, harness.NewMQTT(b))
}

// ---------------------------------------------------------------------------
// C. Concurrent Load (parallel goroutines publishing)
// ---------------------------------------------------------------------------

func benchConcurrent(b *testing.B, h *harness.TransportHarness, goroutines int) {
	b.Helper()
	defer h.Teardown()

	var received int64
	h.Handler = func(_ transport.Command) {
		atomic.AddInt64(&received, 1)
	}

	b.ReportAllocs()
	b.ResetTimer()

	var wg sync.WaitGroup
	perGoroutine := b.N / goroutines
	remainder := b.N % goroutines

	for g := 0; g < goroutines; g++ {
		count := perGoroutine
		if g < remainder {
			count++
		}
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for i := 0; i < n; i++ {
				h.Pub.Publish(transport.Command{DirectionUp: i%2 == 0})
			}
		}(count)
	}
	wg.Wait()

	// Wait for all messages to arrive.
	deadline := time.After(30 * time.Second)
	for atomic.LoadInt64(&received) < int64(b.N) {
		select {
		case <-deadline:
			b.Fatalf("concurrent: received %d/%d", atomic.LoadInt64(&received), b.N)
		default:
			time.Sleep(time.Millisecond)
		}
	}

	b.StopTimer()
}

func BenchmarkConcurrent_GRPC(b *testing.B) {
	for _, g := range []int{1, 2, 4, 8, 16} {
		b.Run(fmt.Sprintf("goroutines-%d", g), func(b *testing.B) {
			benchConcurrent(b, harness.NewGRPC(b), g)
		})
	}
}

func BenchmarkConcurrent_WS(b *testing.B) {
	for _, g := range []int{1, 2, 4, 8, 16} {
		b.Run(fmt.Sprintf("goroutines-%d", g), func(b *testing.B) {
			benchConcurrent(b, harness.NewWS(b), g)
		})
	}
}

func BenchmarkConcurrent_MQTT(b *testing.B) {
	for _, g := range []int{1, 2, 4, 8, 16} {
		b.Run(fmt.Sprintf("goroutines-%d", g), func(b *testing.B) {
			benchConcurrent(b, harness.NewMQTT(b), g)
		})
	}
}

// ---------------------------------------------------------------------------
// D. Connection Setup/Teardown
// ---------------------------------------------------------------------------

func BenchmarkConnect_GRPC(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		port, err := harness.FreePort()
		if err != nil {
			b.Fatal(err)
		}
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		pub, err := grpctransport.NewPublisher(addr)
		if err != nil {
			b.Fatal(err)
		}
		sub := grpctransport.NewSubscriber(addr)
		ctx, cancel := context.WithCancel(context.Background())
		go func() { _ = sub.Listen(ctx, func(_ transport.Command) {}) }()
		time.Sleep(10 * time.Millisecond)
		cancel()
		pub.Stop()
	}
}

func BenchmarkConnect_WS(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		port, err := harness.FreePort()
		if err != nil {
			b.Fatal(err)
		}
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		pub, err := wstransport.NewPublisher(addr)
		if err != nil {
			b.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
		sub := wstransport.NewSubscriber(fmt.Sprintf("ws://%s/ws", addr))
		ctx, cancel := context.WithCancel(context.Background())
		go func() { _ = sub.Listen(ctx, func(_ transport.Command) {}) }()
		time.Sleep(10 * time.Millisecond)
		cancel()
		pub.Stop()
	}
}

func BenchmarkConnect_MQTT(b *testing.B) {
	harness.SkipIfNoMQTT(b)
	broker := "tcp://127.0.0.1:1883"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		pub, err := harness.NewMQTTPublisher(broker)
		if err != nil {
			b.Fatal(err)
		}
		sub := harness.NewMQTTSubscriber(broker)
		ctx, cancel := context.WithCancel(context.Background())
		go func() { _ = sub.Listen(ctx, func(_ transport.Command) {}) }()
		time.Sleep(10 * time.Millisecond)
		cancel()
		pub.Stop()
	}
}

// ---------------------------------------------------------------------------
// E. Serialization Overhead
// ---------------------------------------------------------------------------

func BenchmarkSerialize_Protobuf(b *testing.B) {
	msg := &pb.MasterMessage{DirectionUp: true}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := proto.Marshal(msg)
		if err != nil {
			b.Fatal(err)
		}
		var out pb.MasterMessage
		if err := proto.Unmarshal(data, &out); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSerialize_JSON(b *testing.B) {
	type wsMsg struct {
		DirectionUp bool `json:"direction_up"`
	}
	msg := wsMsg{DirectionUp: true}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := json.Marshal(msg)
		if err != nil {
			b.Fatal(err)
		}
		var out wsMsg
		if err := json.Unmarshal(data, &out); err != nil {
			b.Fatal(err)
		}
	}
}
