package slave

import "flag"

// Config — конфигурация slave-сервиса.
type Config struct {
	MasterAddr  string // Адрес master-сервера (gRPC)
	Transport   string // Транспорт: grpc / websocket / mqtt
	MasterWSURL string // WebSocket URL мастера
	MQTTBroker  string // Адрес MQTT-брокера
}

// LoadConfig загружает конфигурацию из флагов командной строки.
func LoadConfig() Config {
	cfg := Config{}
	flag.StringVar(&cfg.MasterAddr, "master-addr", "192.168.50.127:50051", "Master gRPC server address")
	flag.StringVar(&cfg.Transport, "transport", "grpc", "Transport: grpc / websocket / mqtt")
	flag.StringVar(&cfg.MasterWSURL, "master-ws-url", "ws://192.168.50.127:8080/ws", "Master WebSocket URL")
	flag.StringVar(&cfg.MQTTBroker, "mqtt-broker", "tcp://localhost:1883", "MQTT broker address")
	flag.Parse()
	return cfg
}
