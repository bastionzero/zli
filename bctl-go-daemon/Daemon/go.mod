module bastionzero.com/bctl/v1/Daemon

go 1.16

replace bastionzero.com/bctl/v1/commonWebsocketClient => ../CommonWebsocketClient

require (
	bastionzero.com/bctl/v1/commonWebsocketClient v0.0.0-00010101000000-000000000000 // indirect
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	k8s.io/apimachinery v0.21.3 // indirect
)
