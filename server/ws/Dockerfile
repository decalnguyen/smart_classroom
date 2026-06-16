# Pinned builder; static binary; minimal non-root runtime (hardened).
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
ENV GOPROXY=direct
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build ./server/ws

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /build /build
USER nonroot:nonroot
EXPOSE 8082
ENTRYPOINT ["/build"]
