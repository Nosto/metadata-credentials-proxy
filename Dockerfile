FROM ubuntu:xenial

RUN apt-get update && apt-get install -y software-properties-common
RUN add-apt-repository ppa:gophers/archive

RUN apt-get update && apt-get install -y golang-1.8 make git docker.io dnsutils net-tools dnsutils
RUN ln -sf /usr/lib/go-1.8/bin/go /usr/bin/go

RUN mkdir /opt/metadata
COPY Makefile metadata_wrapper_linux.sh /opt/metadata/
WORKDIR /opt/metadata
RUN make deps

COPY main.go /opt/metadata/

RUN make build && cp metadata /usr/bin/
