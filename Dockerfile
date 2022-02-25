FROM debian:bullseye-backports

RUN apt-get update && apt-get install -y golang-1.16-go make git docker.io dnsutils net-tools dnsutils

RUN mkdir /opt/metadata
COPY Makefile metadata_wrapper_linux.sh /opt/metadata/
WORKDIR /opt/metadata

COPY main.go go.mod go.sum /opt/metadata/

RUN make build && cp metadata /usr/bin/
