package harness

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/novozhenin/practic/internal/transport"
	"github.com/novozhenin/practic/internal/transport/grpctransport"
	"github.com/novozhenin/practic/internal/transport/wstransport"
)

// TransportHarness provides unified setup/teardown for benchmarking any transport.
type TransportHarness struct {
	Name    string
	Pub     transport.Publisher
	Handler func(transport.Command) // set by caller before use
	cancel  context.CancelFunc
	stop    func() // publisher-specific cleanup
	ready   int32  // warmup flag
}

// NewGRPC creates a harness backed by gRPC transport.
func NewGRPC(tb testing.TB) *TransportHarness {
	tb.Helper()
	port, err := FreePort()
	if err != nil {
		tb.Fatal(err)
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	pub, err := grpctransport.NewPublisher(addr)
	if err != nil {
		tb.Fatal(err)
	}

	h := &TransportHarness{
		Name: "GRPC",
		Pub:  pub,
		stop: pub.Stop,
	}
	h.startSubscriber(tb, func() transport.Subscriber {
		return grpctransport.NewSubscriber(addr)
	})
	return h
}

// NewWS creates a harness backed by WebSocket transport.
func NewWS(tb testing.TB) *TransportHarness {
	tb.Helper()
	port, err := FreePort()
	if err != nil {
		tb.Fatal(err)
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	pub, err := wstransport.NewPublisher(addr)
	if err != nil {
		tb.Fatal(err)
	}

	// Give HTTP server a moment to start.
	time.Sleep(20 * time.Millisecond)

	h := &TransportHarness{
		Name: "WS",
		Pub:  pub,
		stop: pub.Stop,
	}
	h.startSubscriber(tb, func() transport.Subscriber {
		return wstransport.NewSubscriber(fmt.Sprintf("ws://%s/ws", addr))
	})
	return h
}

// NewMQTT creates a harness backed by MQTT transport.
// Skips the benchmark if the broker is not available.
func NewMQTT(tb testing.TB) *TransportHarness {
	tb.Helper()
	SkipIfNoMQTT(tb)

	broker := "tcp://127.0.0.1:1883"

	pub, err := NewMQTTPublisher(broker)
	if err != nil {
		tb.Fatal(err)
	}

	h := &TransportHarness{
		Name: "MQTT",
		Pub:  pub,
		stop: pub.Stop,
	}
	h.startSubscriber(tb, func() transport.Subscriber {
		return NewMQTTSubscriber(broker)
	})
	return h
}

// Teardown stops subscriber and publisher.
func (h *TransportHarness) Teardown() {
	if h.cancel != nil {
		h.cancel()
	}
	if h.stop != nil {
		h.stop()
	}
}

// startSubscriber launches Listen in a goroutine and waits for a warmup message.
func (h *TransportHarness) startSubscriber(tb testing.TB, newSub func() transport.Subscriber) {
	tb.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	h.cancel = cancel

	sub := newSub()

	// Handler dispatches to the caller-provided handler.
	go func() {
		_ = sub.Listen(ctx, func(cmd transport.Command) {
			// Signal ready on first message.
			atomic.StoreInt32(&h.ready, 1)

			if h.Handler != nil {
				h.Handler(cmd)
			}
		})
	}()

	// Wait for subscriber to connect, then send warmup message.
	time.Sleep(50 * time.Millisecond)
	h.Pub.Publish(transport.Command{DirectionUp: true})

	// Wait until warmup message is delivered.
	deadline := time.After(5 * time.Second)
	for atomic.LoadInt32(&h.ready) == 0 {
		select {
		case <-deadline:
			tb.Fatal("harness: warmup timed out")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}
