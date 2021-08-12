module bastionzero.com/bctl/v1/bctl

go 1.16

replace bastionzero.com/bctl/v1/bzerolib => ../bzerolib

replace bastionzero.com/bctl/v1/Server => ../Server

replace bastionzero.com/bctl/v1/commonWebsocketClient => ../CommonWebsocketClient

require (
	bastionzero.com/bctl/v1/Server v0.0.0-00010101000000-000000000000
	bastionzero.com/bctl/v1/bzerolib v0.0.0-00010101000000-000000000000
	k8s.io/api v0.21.3 // indirect
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v0.21.3
)
