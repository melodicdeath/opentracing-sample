#!/bin/bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bin/gin-sample-tracing ./cmd/gin-sample/main.go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bin/gin-sample-grpc-server ./cmd/grpc-server/main.go

docker build -f build/Dockerfile-gin-sample -t gin-sample-tracing bin/
docker build -f build/Dockerfile-grpc-server -t gin-sample-grpc-server bin/

docker save -o bin/gin-sample-tracing.img gin-sample-tracing
docker save -o bin/gin-sample-grpc-server.img gin-sample-grpc-server

scp -r bin/gin-sample-grpc-server.img bin/gin-sample-tracing.img root@200.200.200.9:/tmp
ssh root@200.200.200.9 -- "docker load < /tmp/gin-sample-tracing.img;docker load < /tmp/gin-sample-grpc-server.img;"
