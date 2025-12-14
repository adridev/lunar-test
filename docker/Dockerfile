FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache make

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make generate
RUN go build -o bin/rockets ./cmd

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/bin/rockets .

EXPOSE 8088

CMD ["./rockets"]
