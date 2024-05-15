FROM golang:1.21.0-bullseye as builder
ARG GO111MODULE=on
ARG GOPATH=/root/go
# temp sqlite3 error fix
#ARG CGO_CFLAGS "-g -O2 -Wno-return-local-addr"

RUN mkdir -p /root/go/src
COPY dyndns /root/go/src/dyndns
WORKDIR /root/go/src/dyndns
RUN go mod tidy
RUN GOOS=linux go build -o /root/go/bin/dyndns && go test -v

FROM debian:11-slim
ARG S6_OVERLAY_VERSION=3.1.6.2

RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
        apt-get install -q -y bind9 dnsutils curl libc6 xz-utils && \
        apt-get clean

ADD https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-noarch.tar.xz /tmp
RUN tar -C / -Jxpf /tmp/s6-overlay-noarch.tar.xz
ADD https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-x86_64.tar.xz /tmp
RUN tar -C / -Jxpf /tmp/s6-overlay-x86_64.tar.xz
COPY deployment/s6-overlay /etc/s6-overlay

# exit when setup script fails
ENV S6_BEHAVIOUR_IF_STAGE2_FAILS=2

RUN chmod 770 /var/cache/bind
COPY deployment/named.conf.options /etc/bind/named.conf.options

WORKDIR /root
COPY --from=builder /root/go/bin/dyndns /root/dyndns
COPY dyndns/views /root/views
COPY dyndns/static /root/static

EXPOSE 53 8080
ENTRYPOINT ["/init"]
