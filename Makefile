.PHONY: proto build-master build-slave build-all

proto:
	protoc --go_out=. --go-grpc_out=. proto/audio.proto proto/control.proto

build-master:
	GOOS=linux GOARCH=arm64 go build -o bin/master ./cmd/master

build-slave:
	GOOS=linux GOARCH=arm GOARM=7 go build -o bin/slave ./cmd/slave

build-all: build-master build-slave
