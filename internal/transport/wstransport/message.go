package wstransport

// wsMessage — JSON-формат сообщения для WebSocket.
type wsMessage struct {
	DirectionUp bool `json:"direction_up"`
}
