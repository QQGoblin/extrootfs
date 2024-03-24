#!/bin/bash

GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bin/extrootfs ./cmd/main.go
docker build \
--build-arg HTTPS_PROXY=http://172.28.117.33:10778 \
--build-arg HTTP_PROXY=http://172.28.117.33:10778 \
-t registry.lqingcloud.cn/develop/extrootfs:latest -f hack/Dockerfile.openeuler .
