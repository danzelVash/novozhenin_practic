package cable

import (
	"context"
	"log"
	"net"
	"strings"
	"time"
)

type SlaveConfig struct {
	ListenAddr string
	OnCommand  func(command string) error
}

func RunSlave(ctx context.Context, cfg SlaveConfig) error {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = DefaultListenAddr
	}

	listenAddr, err := net.ResolveUDPAddr("udp", cfg.ListenAddr)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Printf("[slave/cable] listening on %s", cfg.ListenAddr)

	var lastSeq uint64
	virtualState := false
	buffer := make([]byte, 2048)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return err
		}

		packet, err := UnmarshalPacket(buffer[:n])
		if err != nil {
			log.Printf("[slave/cable] invalid packet from %s: %v", remoteAddr, err)
			continue
		}

		if warning := SequenceWarning(lastSeq, packet.Seq); warning != "" {
			log.Printf("[slave/cable] %s", warning)
		}
		lastSeq = packet.Seq

		delayMS := DelayMillis(time.Now(), packet)
		log.Printf("[slave/cable] VL:%d SEQ:%d CMD:%s delay:%d ms",
			packet.VLID, packet.Seq, packet.Command, delayMS)

		// Виртуальное состояние исполняет команду без зависимости от GPIO.
		virtualState = strings.EqualFold(packet.Command, CommandUp)
		log.Printf("[slave/cable] virtual_state=%t", virtualState)

		if cfg.OnCommand != nil {
			if err := cfg.OnCommand(packet.Command); err != nil {
				log.Printf("[slave/cable] command execution error: %v", err)
				continue
			}
		}

		if _, err := conn.WriteToUDP([]byte(AckMessage), remoteAddr); err != nil {
			log.Printf("[slave/cable] ack send error: %v", err)
			continue
		}
		log.Printf("[slave/cable] ACK sent to %s", remoteAddr)
	}
}
