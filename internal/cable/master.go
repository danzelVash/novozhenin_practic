package cable

import (
	"context"
	"log"
	"net"
	"time"
)

type MasterConfig struct {
	SlaveAddr string
}

func RunMaster(ctx context.Context, cfg MasterConfig) error {
	if cfg.SlaveAddr == "" {
		cfg.SlaveAddr = DefaultSlaveAddr
	}

	remoteAddr, err := net.ResolveUDPAddr("udp", cfg.SlaveAddr)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Printf("[master/cable] deterministic UDP scheduler started, slave=%s bag=%s vl_id=%d",
		cfg.SlaveAddr, DefaultBAG, DefaultVLID)

	go readACKs(ctx, conn)

	start := time.Now()
	nextTx := start
	var seq uint64 = 1

	for {
		if err := waitUntil(ctx, nextTx); err != nil {
			return err
		}

		packet := Packet{
			VLID:      DefaultVLID,
			Seq:       seq,
			Timestamp: time.Now().UnixMilli(),
			Command:   CommandForSeq(seq),
		}

		payload, err := packet.Marshal()
		if err != nil {
			return err
		}
		if _, err := conn.Write(payload); err != nil {
			return err
		}

		log.Printf("[master/cable] sent VL:%d SEQ:%d CMD:%s ts=%d",
			packet.VLID, packet.Seq, packet.Command, packet.Timestamp)

		seq++
		// BAG: фиксированный интервал между передачами без накопления дрейфа.
		nextTx = nextTx.Add(DefaultBAG)
	}
}

func readACKs(ctx context.Context, conn *net.UDPConn) {
	buffer := make([]byte, 64)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn.Read(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			log.Printf("[master/cable] ack read error: %v", err)
			return
		}

		log.Printf("[master/cable] received %s", string(buffer[:n]))
	}
}

func waitUntil(ctx context.Context, target time.Time) error {
	delay := time.Until(target)
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
