# Practic — управление сервоприводом по голосу

Master записывает аудио, определяет команду через нейросеть и отправляет её на slave, который управляет сервоприводом.

## Транспорты

Поддерживаются три транспорта, переключаемые флагом `-transport`:

| Транспорт | Описание |
|-----------|----------|
| `grpc` (по умолчанию) | gRPC bidirectional stream |
| `websocket` | WebSocket (gorilla/websocket) |
| `mqtt` | MQTT (eclipse paho) |

## Сборка

```bash
# Все бинарники (master + slave)
make build-all

# По отдельности
make master_grpc
make master_websocket
make master_mqtt
make slave_grpc
make slave_websocket
make slave_mqtt
```

Master собирается под `linux/arm64`, slave — под `linux/arm/v7`.

## Запуск

### gRPC (по умолчанию)

```bash
# Master
./master -transport grpc -grpc-port :50051

# Slave
./slave -transport grpc -master-addr 192.168.50.127:50051
```

### WebSocket

```bash
# Master
./master -transport websocket -ws-port :8080

# Slave
./slave -transport websocket -master-ws-url ws://192.168.50.127:8080/ws
```

### MQTT

Требуется MQTT-брокер (например mosquitto).

```bash
# Master
./master -transport mqtt -mqtt-broker tcp://localhost:1883

# Slave
./slave -transport mqtt -mqtt-broker tcp://localhost:1883
```

## Флаги

### Master

| Флаг | По умолчанию | Описание |
|------|-------------|----------|
| `-transport` | `grpc` | Транспорт: grpc / websocket / mqtt |
| `-grpc-port` | `:50051` | Порт gRPC-сервера |
| `-ws-port` | `:8080` | Порт WebSocket-сервера |
| `-mqtt-broker` | `tcp://localhost:1883` | Адрес MQTT-брокера |
| `-audio-device` | `plughw:2,0` | ALSA-устройство |
| `-audio-rate` | `16000` | Частота дискретизации |
| `-vad-threshold` | `0.08` | Порог VAD |
| `-silence-dur` | `1.5` | Длительность тишины (сек) |
| `-neuro-addr` | `192.168.50.96:8000` | Адрес нейросервиса |

### Slave

| Флаг | По умолчанию | Описание |
|------|-------------|----------|
| `-transport` | `grpc` | Транспорт: grpc / websocket / mqtt |
| `-master-addr` | `192.168.50.127:50051` | Адрес master (gRPC) |
| `-master-ws-url` | `ws://192.168.50.127:8080/ws` | URL master (WebSocket) |
| `-mqtt-broker` | `tcp://localhost:1883` | Адрес MQTT-брокера |
