FROM debian:buster-backports

RUN apt-get update && apt-get install -y golang-1.14 make git docker.io dnsutils net-tools dnsutils

RUN mkdir /opt/metadata
COPY Makefile metadata_wrapper_linux.sh /opt/metadata/
WORKDIR /opt/metadata
RUN make deps

COPY main.go /opt/metadata/

RUN make build && cp metadata /usr/bin/
