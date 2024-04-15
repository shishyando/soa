FROM golang:1.22-alpine

WORKDIR /main_user_service
COPY go.* .

RUN go mod download -x

COPY * .

RUN go build .

ENTRYPOINT ["./main_user_service", "--private", "./private_unencrypted.pem", "--public", "./public.pem"]

