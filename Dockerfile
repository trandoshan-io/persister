# build image
FROM golang:1.13.1-alpine3.10 as builder

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh

RUN go get -v github.com/nats-io/nats.go/ && \
    go get -v go.mongodb.org/mongo-driver/bson && \ 
    go get -v go.mongodb.org/mongo-driver/mongo && \
    go get -v go.mongodb.org/mongo-driver/mongo/options && \
    go get -v go.mongodb.org/mongo-driver/mongo/readpref

COPY . /app/
WORKDIR /app

# Test then build app
RUN CGO_ENABLED=0 go test -v
RUN go build -v persister.go


# runtime image
FROM alpine:latest
COPY --from=builder /app/persister /app/

WORKDIR /app/
CMD ["./persister"]