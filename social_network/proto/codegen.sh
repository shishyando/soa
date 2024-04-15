#!/bin/bash 

protoc --go_out=. --go_opt=paths=source_relative \                                                                                                                       social_network/proto (hw3) shishyando-osx
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  post.proto
