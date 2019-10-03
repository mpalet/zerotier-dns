ARG GO_VERSION=1.13
FROM golang:${GO_VERSION} AS builder

RUN useradd ztdns

WORKDIR /go/src/github.com/mje-nz/ztdns

# Fetch and cache dependencies
COPY ./go.mod ./go.sum ./
RUN go mod download

# Build static binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go install -ldflags="-w -s"


FROM scratch

# We need ca-certificates for HTTPS, /etc/passwd to log in, and zoneinfo for
# time zones
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

WORKDIR /app
COPY --from=builder /go/bin/ztdns .

USER ztdns

ENTRYPOINT ["./ztdns"]
CMD ["server"]
EXPOSE 53/udp
