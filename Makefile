.PHONY: proto build-master build-slave build-all \
       master_grpc master_websocket master_mqtt \
       slave_grpc slave_websocket slave_mqtt

proto:
	protoc --go_out=. --go-grpc_out=. proto/audio.proto proto/control.proto

build-master:
	GOOS=linux GOARCH=arm64 go build -o bin/master ./cmd/master

build-slave:
	GOOS=linux GOARCH=arm GOARM=7 go build -o bin/slave ./cmd/slave

build-all: build-master build-slave

master_grpc:
	GOOS=linux GOARCH=arm64 go build -o bin/master_grpc ./cmd/master

master_websocket:
	GOOS=linux GOARCH=arm64 go build -o bin/master_websocket ./cmd/master

master_mqtt:
	GOOS=linux GOARCH=arm64 go build -o bin/master_mqtt ./cmd/master

slave_grpc:
	GOOS=linux GOARCH=arm GOARM=7 go build -o bin/slave_grpc ./cmd/slave

slave_websocket:
	GOOS=linux GOARCH=arm GOARM=7 go build -o bin/slave_websocket ./cmd/slave

slave_mqtt:
	GOOS=linux GOARCH=arm GOARM=7 go build -o bin/slave_mqtt ./cmd/slave
