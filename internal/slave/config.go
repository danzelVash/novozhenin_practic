package slave

import "flag"

// Config — конфигурация slave-сервиса.
type Config struct {
	MasterAddr string // Адрес master-сервера
}

// LoadConfig загружает конфигурацию из флагов командной строки.
func LoadConfig() Config {
	cfg := Config{}
	flag.StringVar(&cfg.MasterAddr, "master-addr", "192.168.50.127:50051", "Master gRPC server address")
	flag.Parse()
	return cfg
}
