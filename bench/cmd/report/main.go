package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/novozhenin/practic/bench/harness"
	"github.com/novozhenin/practic/internal/transport"
	"github.com/novozhenin/practic/internal/transport/grpctransport"
	"github.com/novozhenin/practic/internal/transport/wstransport"
	pb "github.com/novozhenin/practic/pkg/pb"
	"google.golang.org/protobuf/proto"
)

const (
	messageCount  = 10_000
	connectIter   = 100
	serializeIter = 100_000
)

type result struct {
	Name       string
	Throughput float64 // msgs/sec
	AvgLatency time.Duration
	P50        time.Duration
	P95        time.Duration
	P99        time.Duration
	Max        time.Duration
	ConnSetup  time.Duration // avg
}

func main() {
	log.SetOutput(io.Discard)

	fmt.Println("=== Transport Benchmark Report ===")
	fmt.Printf("Messages per test: %d\n", messageCount)
	fmt.Printf("Connection iterations: %d\n\n", connectIter)

	var results []result

	// gRPC
	if r, err := benchTransport("gRPC", setupGRPC); err != nil {
		fmt.Fprintf(os.Stderr, "gRPC error: %v\n", err)
	} else {
		results = append(results, r)
	}

	// WebSocket
	if r, err := benchTransport("WebSocket", setupWS); err != nil {
		fmt.Fprintf(os.Stderr, "WebSocket error: %v\n", err)
	} else {
		results = append(results, r)
	}

	// MQTT
	if harness.MQTTAvailable() {
		if r, err := benchTransport("MQTT", setupMQTT); err != nil {
			fmt.Fprintf(os.Stderr, "MQTT error: %v\n", err)
		} else {
			results = append(results, r)
		}
	} else {
		fmt.Println("[MQTT] SKIPPED — broker not available on localhost:1883")
		fmt.Println()
	}

	// Print throughput & latency table.
	fmt.Println("--- Throughput & Latency ---")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "Transport\tThroughput (msg/s)\tAvg Latency\tp50\tp95\tp99\tmax\t")
	for _, r := range results {
		fmt.Fprintf(w, "%s\t%.0f\t%v\t%v\t%v\t%v\t%v\t\n",
			r.Name, r.Throughput, r.AvgLatency.Truncate(time.Microsecond),
			r.P50.Truncate(time.Microsecond),
			r.P95.Truncate(time.Microsecond),
			r.P99.Truncate(time.Microsecond),
			r.Max.Truncate(time.Microsecond))
	}
	w.Flush()
	fmt.Println()

	// Print connection setup table.
	fmt.Println("--- Connection Setup (avg) ---")
	w = tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "Transport\tAvg Setup Time\t")
	for _, r := range results {
		fmt.Fprintf(w, "%s\t%v\t\n", r.Name, r.ConnSetup.Truncate(time.Microsecond))
	}
	w.Flush()
	fmt.Println()

	// Serialization comparison.
	fmt.Println("--- Serialization Overhead ---")
	protoD := benchProtobuf()
	jsonD := benchJSON()
	w = tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "Format\tAvg Marshal+Unmarshal\tOps/sec\t")
	fmt.Fprintf(w, "Protobuf\t%v\t%.0f\t\n", protoD.Truncate(time.Nanosecond), float64(time.Second)/float64(protoD))
	fmt.Fprintf(w, "JSON\t%v\t%.0f\t\n", jsonD.Truncate(time.Nanosecond), float64(time.Second)/float64(jsonD))
	w.Flush()
}

type setupFunc func() (pub transport.Publisher, stop func(), sub transport.Subscriber, err error)

func benchTransport(name string, setup setupFunc) (result, error) {
	r := result{Name: name}

	// --- Latency distribution ---
	pub, stop, sub, err := setup()
	if err != nil {
		return r, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan time.Time, 1)

	go func() {
		_ = sub.Listen(ctx, func(_ transport.Command) {
			ch <- time.Now()
		})
	}()
	time.Sleep(100 * time.Millisecond)

	// Warmup.
	pub.Publish(transport.Command{DirectionUp: true})
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		cancel()
		stop()
		return r, fmt.Errorf("%s: warmup timeout", name)
	}

	durations := make([]time.Duration, 0, messageCount)
	for i := 0; i < messageCount; i++ {
		start := time.Now()
		pub.Publish(transport.Command{DirectionUp: i%2 == 0})
		select {
		case end := <-ch:
			durations = append(durations, end.Sub(start))
		case <-time.After(10 * time.Second):
			cancel()
			stop()
			return r, fmt.Errorf("%s: message %d timeout", name, i)
		}
	}

	cancel()
	stop()

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	n := len(durations)

	var total time.Duration
	for _, d := range durations {
		total += d
	}
	r.AvgLatency = total / time.Duration(n)
	r.Throughput = float64(n) / total.Seconds()
	r.P50 = durations[n*50/100]
	r.P95 = durations[n*95/100]
	r.P99 = durations[n*99/100]
	r.Max = durations[n-1]

	fmt.Printf("[%s] latency done\n", name)

	// --- Connection setup ---
	r.ConnSetup = benchConnect(name, setup)
	fmt.Printf("[%s] connect done\n", name)

	return r, nil
}

func benchConnect(name string, setup setupFunc) time.Duration {
	var total time.Duration
	for i := 0; i < connectIter; i++ {
		start := time.Now()
		_, stop, sub, err := setup()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s connect iter %d: %v\n", name, i, err)
			continue
		}
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			_ = sub.Listen(ctx, func(_ transport.Command) {})
		}()
		time.Sleep(20 * time.Millisecond)
		cancel()
		stop()
		total += time.Since(start)
	}
	return total / time.Duration(connectIter)
}

func setupGRPC() (transport.Publisher, func(), transport.Subscriber, error) {
	port, err := harness.FreePort()
	if err != nil {
		return nil, nil, nil, err
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	pub, err := grpctransport.NewPublisher(addr)
	if err != nil {
		return nil, nil, nil, err
	}
	sub := grpctransport.NewSubscriber(addr)
	return pub, pub.Stop, sub, nil
}

func setupWS() (transport.Publisher, func(), transport.Subscriber, error) {
	port, err := harness.FreePort()
	if err != nil {
		return nil, nil, nil, err
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	pub, err := wstransport.NewPublisher(addr)
	if err != nil {
		return nil, nil, nil, err
	}
	time.Sleep(20 * time.Millisecond)
	sub := wstransport.NewSubscriber(fmt.Sprintf("ws://%s/ws", addr))
	return pub, pub.Stop, sub, nil
}

func setupMQTT() (transport.Publisher, func(), transport.Subscriber, error) {
	broker := "tcp://127.0.0.1:1883"
	pub, err := harness.NewMQTTPublisher(broker)
	if err != nil {
		return nil, nil, nil, err
	}
	sub := harness.NewMQTTSubscriber(broker)
	return pub, pub.Stop, sub, nil
}

func benchProtobuf() time.Duration {
	msg := &pb.MasterMessage{DirectionUp: true}
	start := time.Now()
	for i := 0; i < serializeIter; i++ {
		data, _ := proto.Marshal(msg)
		var out pb.MasterMessage
		_ = proto.Unmarshal(data, &out)
	}
	return time.Since(start) / time.Duration(serializeIter)
}

func benchJSON() time.Duration {
	type wsMsg struct {
		DirectionUp bool `json:"direction_up"`
	}
	msg := wsMsg{DirectionUp: true}
	start := time.Now()
	for i := 0; i < serializeIter; i++ {
		data, _ := json.Marshal(msg)
		var out wsMsg
		_ = json.Unmarshal(data, &out)
	}
	return time.Since(start) / time.Duration(serializeIter)
}
