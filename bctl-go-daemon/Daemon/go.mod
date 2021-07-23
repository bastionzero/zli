module bastionzero.com/bctl/v1/Daemon

go 1.16

replace bastionzero.com/bctl/v1/commonWebsocketClient => ../CommonWebsocketClient

require (
	bastionzero.com/bctl/v1/commonWebsocketClient v0.0.0-00010101000000-000000000000
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/websocket v1.4.2
	k8s.io/apimachinery v0.21.3
)
