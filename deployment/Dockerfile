FROM golang:1.22-alpine as builder
ARG CGO_ENABLED=1
ARG GOPATH=/root/go

RUN apk add --no-cache gcc musl-dev

RUN mkdir -p /root/go/src
COPY dyndns /root/go/src/dyndns
WORKDIR /root/go/src/dyndns
RUN go mod tidy
RUN go build -o /root/go/bin/dyndns && go test -v

FROM alpine:3.23
ARG S6_OVERLAY_VERSION=3.2.1.0

RUN apk add --no-cache bind bind-tools curl

ADD https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-noarch.tar.xz /tmp
RUN tar -C / -Jxpf /tmp/s6-overlay-noarch.tar.xz
ADD https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-x86_64.tar.xz /tmp
RUN tar -C / -Jxpf /tmp/s6-overlay-x86_64.tar.xz
COPY deployment/s6-overlay /etc/s6-overlay

# exit when setup script fails
ENV S6_BEHAVIOUR_IF_STAGE2_FAILS=2

COPY deployment/named.conf /etc/bind/named.conf
# RUN cat /etc/bind/named.conf.authoritative - <<<'include "/etc/bind/named.conf.ddns";' >/etc/bind/named.conf

WORKDIR /root
COPY --from=builder /root/go/bin/dyndns dyndns
COPY dyndns/views views
COPY dyndns/static static

EXPOSE 53 8080
ENTRYPOINT ["/init"]
