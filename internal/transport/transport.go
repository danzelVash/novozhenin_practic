package transport

import "context"

// Command — команда управления сервоприводом.
type Command struct {
	DirectionUp bool
}

// Publisher — отправка команд slave-устройствам (master-сторона).
// Реализации: grpctransport, в будущем — websocket, MQTT.
type Publisher interface {
	Publish(cmd Command)
}

// Subscriber — получение команд от master (slave-сторона).
// Реализации: grpctransport, в будущем — websocket, MQTT.
type Subscriber interface {
	Listen(ctx context.Context, handler func(Command)) error
}
