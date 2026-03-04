package harness

import (
	"net"
	"testing"
	"time"
)

const defaultMQTTAddr = "127.0.0.1:1883"

// MQTTAvailable checks whether an MQTT broker is listening on localhost:1883.
func MQTTAvailable() bool {
	conn, err := net.DialTimeout("tcp", defaultMQTTAddr, 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// SkipIfNoMQTT skips the current test/benchmark if MQTT broker is not available.
func SkipIfNoMQTT(tb testing.TB) {
	tb.Helper()
	if !MQTTAvailable() {
		tb.Skip("MQTT broker not available on localhost:1883, skipping")
	}
}
