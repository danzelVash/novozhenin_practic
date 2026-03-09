package cable

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	DefaultVLID       = 1
	DefaultBAG        = 7 * time.Second
	DefaultSlaveAddr  = "192.168.10.2:9000"
	DefaultListenAddr = ":9000"
	CommandUp         = "UP"
	CommandDown       = "DOWN"
	AckMessage        = "ACK"
)

// Packet имитирует AFDX Virtual Link поверх UDP/JSON.
// VL_ID, SEQ и TIMESTAMP передаются в каждом кадре как метаданные канала.
type Packet struct {
	VLID      int    `json:"vl_id"`
	Seq       uint64 `json:"seq"`
	Timestamp int64  `json:"timestamp"`
	Command   string `json:"command"`
}

func (p Packet) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func UnmarshalPacket(data []byte) (Packet, error) {
	var packet Packet
	if err := json.Unmarshal(data, &packet); err != nil {
		return Packet{}, err
	}
	if packet.VLID == 0 {
		return Packet{}, fmt.Errorf("vl_id is required")
	}
	if packet.Seq == 0 {
		return Packet{}, fmt.Errorf("seq must be positive")
	}
	if packet.Timestamp == 0 {
		return Packet{}, fmt.Errorf("timestamp is required")
	}
	if packet.Command != CommandUp && packet.Command != CommandDown {
		return Packet{}, fmt.Errorf("unknown command: %s", packet.Command)
	}
	return packet, nil
}

func CommandForSeq(seq uint64) string {
	if seq%2 == 1 {
		return CommandUp
	}
	return CommandDown
}

func DelayMillis(now time.Time, packet Packet) int64 {
	return now.Sub(time.UnixMilli(packet.Timestamp)).Milliseconds()
}

// SequenceWarning реализует контроль последовательности кадров.
func SequenceWarning(lastSeq uint64, currentSeq uint64) string {
	if lastSeq == 0 || currentSeq == lastSeq+1 {
		return ""
	}
	if currentSeq <= lastSeq {
		return fmt.Sprintf("WARNING: out-of-order packet (last=%d current=%d)", lastSeq, currentSeq)
	}
	return fmt.Sprintf("WARNING: packet loss (expected=%d got=%d)", lastSeq+1, currentSeq)
}
