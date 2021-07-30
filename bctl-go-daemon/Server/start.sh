#!/bin/sh
if [ $DEV == "true" ]; then
    sleep infinity
else
    cd /bctl-server/Server/ && go get k8s.io/apimachinery/pkg/apis/meta/v1@v0.21.3
    cd /bctl-server/Server/ && go run /bctl-server/Server/main.go -serviceURL=$SERVICE_URL
fi