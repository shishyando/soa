FROM golang:1.22-alpine

WORKDIR /social_network
COPY . .

WORKDIR /social_network/post_service

RUN go mod download -x

RUN go build

ENTRYPOINT ["./post_service"]

