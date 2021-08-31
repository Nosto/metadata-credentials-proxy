export PATH := $(PATH):/usr/lib/go-1.15/bin
PACKAGE = metadata

all: build

build:
	go build -o metadata

format:
	gofmt -w *.go
