package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/novozhenin/practic/internal/transport"
)

const benchTopic = "servo/command"

type mqttMsg struct {
	DirectionUp bool `json:"direction_up"`
}

// MQTTPublisher wraps an MQTT client with a unique client ID for benchmarks.
type MQTTPublisher struct {
	client mqtt.Client
}

// NewMQTTPublisher creates a publisher with a unique ID and connects to the broker.
func NewMQTTPublisher(broker string) (*MQTTPublisher, error) {
	id := fmt.Sprintf("bench-pub-%d", rand.Int63())
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(id).
		SetCleanSession(true).
		SetAutoReconnect(false)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("mqtt publisher connect: %w", token.Error())
	}
	return &MQTTPublisher{client: client}, nil
}

// Publish sends a command to the bench topic.
func (p *MQTTPublisher) Publish(cmd transport.Command) {
	data, _ := json.Marshal(mqttMsg{DirectionUp: cmd.DirectionUp})
	token := p.client.Publish(benchTopic, 1, false, data)
	token.Wait()
}

// Stop disconnects the publisher.
func (p *MQTTPublisher) Stop() {
	p.client.Disconnect(250)
}

// MQTTSubscriber wraps an MQTT client with a unique client ID for benchmarks.
type MQTTSubscriber struct {
	broker string
}

// NewMQTTSubscriber creates a subscriber wrapper.
func NewMQTTSubscriber(broker string) *MQTTSubscriber {
	return &MQTTSubscriber{broker: broker}
}

// Listen connects, subscribes and calls handler for each message. Blocks until ctx done.
func (s *MQTTSubscriber) Listen(ctx context.Context, handler func(transport.Command)) error {
	id := fmt.Sprintf("bench-sub-%d", rand.Int63())
	opts := mqtt.NewClientOptions().
		AddBroker(s.broker).
		SetClientID(id).
		SetCleanSession(true).
		SetAutoReconnect(false)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("mqtt subscriber connect: %w", token.Error())
	}
	defer client.Disconnect(250)

	token := client.Subscribe(benchTopic, 1, func(_ mqtt.Client, msg mqtt.Message) {
		var m mqttMsg
		if err := json.Unmarshal(msg.Payload(), &m); err != nil {
			return
		}
		handler(transport.Command{DirectionUp: m.DirectionUp})
	})
	token.Wait()
	if token.Error() != nil {
		return fmt.Errorf("mqtt subscribe: %w", token.Error())
	}

	<-ctx.Done()
	return ctx.Err()
}
