# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /deaddrop ./cmd/deaddrop

# Stage 2: Runtime
FROM alpine:3.20

COPY --from=builder /deaddrop /deaddrop

EXPOSE 8080

ENTRYPOINT ["/deaddrop"]
