ARG GO_VERSION=1.13
FROM golang:${GO_VERSION} AS builder

RUN useradd zerotier-dns

WORKDIR /go/src/github.com/mje-nz/zerotier-dns

# Fetch and cache dependencies
COPY ./go.mod ./go.sum ./
RUN go mod download

# Build static binary and allow it to bind to ports <1000
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go install -ldflags="-w -s" && \
	# NB Only works on BuildKit
	# https://github.com/moby/moby/issues/35699
	setcap cap_net_bind_service=+ep /go/bin/zerotier-dns

FROM scratch

# We need ca-certificates for HTTPS, /etc/passwd to log in, and zoneinfo for
# time zones
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

WORKDIR /app
COPY --from=builder /go/bin/zerotier-dns .
COPY zerotier-dns.example.yml zerotier-dns.yml

USER zerotier-dns

ENTRYPOINT ["./zerotier-dns"]
CMD ["server"]
EXPOSE 53/udp
