FROM golang:1.22-alpine

WORKDIR /social_network
COPY . .

WORKDIR /social_network/main_service

RUN go mod download -x

RUN go build .

ENTRYPOINT ["./main_service", "--private", "./private_unencrypted.pem", "--public", "./public.pem"]

