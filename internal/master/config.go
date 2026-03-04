package master

import "flag"

// Config — конфигурация master-сервиса.
type Config struct {
	AudioDevice  string  // ALSA-устройство (например "plughw:2,0")
	AudioRate    int     // Частота дискретизации (16000)
	VADThreshold float64 // Порог RMS для VAD
	SilenceDur   float64 // Длительность тишины для завершения фразы (сек)
	NeuroAddr    string  // Адрес нейросервиса
	GRPCPort     string  // Порт gRPC-сервера для slave
	Transport    string  // Транспорт: grpc / websocket / mqtt
	WSPort       string  // Порт WebSocket-сервера
	MQTTBroker   string  // Адрес MQTT-брокера
}

// LoadConfig загружает конфигурацию из флагов командной строки.
func LoadConfig() Config {
	cfg := Config{}
	flag.StringVar(&cfg.AudioDevice, "audio-device", "plughw:2,0", "ALSA audio capture device")
	flag.IntVar(&cfg.AudioRate, "audio-rate", 16000, "Audio sample rate in Hz")
	flag.Float64Var(&cfg.VADThreshold, "vad-threshold", 0.08, "VAD RMS threshold (0-1)")
	flag.Float64Var(&cfg.SilenceDur, "silence-dur", 1.5, "Silence duration to end phrase (seconds)")
	flag.StringVar(&cfg.NeuroAddr, "neuro-addr", "192.168.50.96:8000", "Neural network service address")
	flag.StringVar(&cfg.GRPCPort, "grpc-port", ":50051", "gRPC server port for slave")
	flag.StringVar(&cfg.Transport, "transport", "grpc", "Transport: grpc / websocket / mqtt")
	flag.StringVar(&cfg.WSPort, "ws-port", ":8080", "WebSocket server port")
	flag.StringVar(&cfg.MQTTBroker, "mqtt-broker", "tcp://localhost:1883", "MQTT broker address")
	flag.Parse()
	return cfg
}
