ARG GO_VERSION=1.13
FROM golang:${GO_VERSION} AS builder

WORKDIR /go/src/github.com/mje-nz/ztdns

# Fetch and cache dependencies
COPY ./go.mod ./go.sum ./
RUN go mod download

# Build static binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go install


FROM alpine:3.10

# We need to add ca-certificates in order to make HTTPS API calls
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

WORKDIR /app
# Copy binary
COPY --from=builder /go/bin/ztdns .

ENTRYPOINT ["./ztdns"]
CMD ["server"]
EXPOSE 53/udp
