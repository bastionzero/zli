#!/bin/sh
cd /bctl-server/bctl/agent 
go get k8s.io/apimachinery/pkg/apis/meta/v1@v0.21.3
go mod download golang.org/x/sys
go get k8s.io/client-go/rest@v0.21.3
go get k8s.io/client-go/tools/remotecommand@v0.21.3
go get k8s.io/api/core/v1@v0.21.3
go run /bctl-server/bctl/agent/agent.go -serviceUrl=$SERVICE_URL