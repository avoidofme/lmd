FROM golang:alpine AS builder

WORKDIR /app

ENV GOPROXY=direct
ENV GOOS=linux
ENV CGO_ENABLED=false

RUN apk add --no-cache git=~2

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o lmd main.go

FROM alpine:3

RUN addgroup -g 1001 mygroup && \
  adduser -u 1001 -G mygroup -D myuser

WORKDIR /app

RUN chown myuser:mygroup /app

COPY --from=builder --chown=myuser:mygroup /app/lmd .

USER 1001:1001

EXPOSE 8080

CMD [ "./lmd" ]
