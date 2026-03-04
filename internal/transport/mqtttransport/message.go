package mqtttransport

// mqttMessage — JSON-формат сообщения для MQTT.
type mqttMessage struct {
	DirectionUp bool `json:"direction_up"`
}
