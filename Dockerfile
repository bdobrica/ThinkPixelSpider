# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags '-s -w' -o /bin/thinkpixelspider ./cmd/thinkpixelspider
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags '-s -w' -o /bin/thinkpixelspiderd ./cmd/thinkpixelspiderd

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/thinkpixelspider /usr/local/bin/thinkpixelspider
COPY --from=builder /bin/thinkpixelspiderd /usr/local/bin/thinkpixelspiderd

RUN adduser -D -u 1000 spider
USER spider

ENTRYPOINT ["thinkpixelspiderd"]
