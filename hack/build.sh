#!/bin/bash

GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bin/extrootfs ./cmd/main.go
docker build -t registry.lqingcloud.cn/develop/extrootfs:latest -f hack/Dockerfile .
