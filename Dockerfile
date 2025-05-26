FROM golang:latest as builder

WORKDIR /app

COPY go.mod go.sum ./

ENV GOPROXY=direct

RUN go mod download

COPY . .

RUN go build -o build .

FROM ubuntu

RUN apt-get update && apt-get install -y curl

WORKDIR /app

COPY --from=builder /app/build .

ENTRYPOINT ["/app/build"]