Version := $(shell date "+%Y%m%d%H%M")
GitCommit := $(shell git rev-parse HEAD)
DIR := $(shell pwd)
LDFLAGS := "-s -w -X main.Version=$(Version) -X main.GitCommit=$(GitCommit)"

run: build
	./bin/example --shutdown_timeout 5s --load-job

build:
	go build -ldflags $(LDFLAGS) -o bin/example *.go

.PHONY: run build
