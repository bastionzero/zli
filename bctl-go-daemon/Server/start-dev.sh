#!/bin/sh
cd /bctl-server/bctl/agent 
go get k8s.io/apimachinery/pkg/apis/meta/v1@v0.21.3
go mod download golang.org/x/sys
go run /bctl-server/bctl/agent/agent.go -serviceUrl=$SERVICE_URL