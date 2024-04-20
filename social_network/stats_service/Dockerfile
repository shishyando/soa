FROM golang:1.22-alpine

WORKDIR /social_network
COPY . .

WORKDIR /social_network/stats_service

RUN go mod download -x

RUN go build .

ENTRYPOINT ["./stats_service"]

