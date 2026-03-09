package cable

import (
	"testing"
	"time"
)

func TestCommandForSeq(t *testing.T) {
	if got := CommandForSeq(1); got != CommandUp {
		t.Fatalf("seq 1: got %s want %s", got, CommandUp)
	}
	if got := CommandForSeq(2); got != CommandDown {
		t.Fatalf("seq 2: got %s want %s", got, CommandDown)
	}
}

func TestSequenceWarning(t *testing.T) {
	if warning := SequenceWarning(5, 6); warning != "" {
		t.Fatalf("unexpected warning: %s", warning)
	}
	if warning := SequenceWarning(5, 8); warning == "" {
		t.Fatal("expected packet loss warning")
	}
	if warning := SequenceWarning(5, 4); warning == "" {
		t.Fatal("expected out-of-order warning")
	}
}

func TestDelayMillis(t *testing.T) {
	now := time.UnixMilli(2_000)
	packet := Packet{Timestamp: 1_250}

	if got := DelayMillis(now, packet); got != 750 {
		t.Fatalf("got %d want 750", got)
	}
}
