module bastionzero.com/bctl/v1/bctl

go 1.16

replace bastionzero.com/bctl/v1/bzerolib => ../bzerolib

require (
	bastionzero.com/bctl/v1/bzerolib v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.1.2
	github.com/gorilla/websocket v1.4.2
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v0.21.3
)
