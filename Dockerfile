FROM golang:1.13.3-alpine3.10 as builder

RUN apk add curl git gcc musl-dev

WORKDIR /tmp/build/

COPY go.* ./

RUN go mod download

COPY . ./

RUN go build -o /tmp/didiff ./cmd/didiff

FROM alpine

COPY --from=builder /tmp/didiff /usr/local/bin/didiff

ENTRYPOINT ["/usr/local/bin/didiff"]
