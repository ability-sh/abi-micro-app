#!/bin/sh

# brew install protobuf
# go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
# go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

protoc --go-grpc_out=require_unimplemented_servers=false:./pb --go_out=./pb ./pb/*.proto
