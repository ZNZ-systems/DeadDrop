# Stage 1: Build
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /deaddrop ./cmd/deaddrop

# Stage 2: Runtime
FROM alpine:3.20

COPY --from=builder /deaddrop /deaddrop

EXPOSE 8080

ENTRYPOINT ["/deaddrop"]
