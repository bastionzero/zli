#!/bin/sh
<<<<<<< HEAD
cd /bctl-server/Server/ && go get k8s.io/apimachinery/pkg/apis/meta/v1@v0.21.3
cd /bctl-server/Server/ && go run /bctl-server/Server/main.go -serviceURL=$SERVICE_URL
=======
cd /bctl-server-files/bctl/agent 
go get k8s.io/apimachinery/pkg/apis/meta/v1@v0.21.3
go mod download golang.org/x/sys
go get k8s.io/client-go/rest@v0.21.3
go get k8s.io/client-go/tools/remotecommand@v0.21.3
go get k8s.io/api/core/v1@v0.21.3
go run /bctl-server-files/bctl/agent/agent.go -serviceUrl=$SERVICE_URL
>>>>>>> 724999e (Merged list-targets and list-clusters functionality. Fixed filtering for clusters. Added target users for list-targets)
