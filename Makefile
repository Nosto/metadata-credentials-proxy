PACKAGE = metadata

all: deps build

deps:
	GOPATH=$(CURDIR) go get github.com/aws/aws-sdk-go
	GOPATH=$(CURDIR) go get github.com/docker/docker/client
	GOPATH=$(CURDIR) go get golang.org/x/net/context

build:
	GOPATH=$(CURDIR) go build

format:
	gofmt -w *.go
