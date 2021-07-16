package ControlWebsocket

import (
	"bastionzero.com/bctl/v1/CommonWebsocketClient"
)

type ControlWebsocket struct {
	WebsocketClient *CommonWebsocketClient.WebsocketClient

	// These are all the    types of channels we have available
	RequestForServerChan    chan CommonWebsocketClient.RequestForServerSignalRMessage
	RequestForStartExecChan chan CommonWebsocketClient.RequestForStartExecToClusterSingalRMessage
	ExecStdinChannel        chan CommonWebsocketClient.SendStdinToClusterSignalRMessage
}

// Constructor to create a new Control Websocket Client
func NewControlWebsocketClient(sessionId string, authHeader string, serviceURL string) {
	ret := ControlWebsocket{}

	// Create our headers and params
	// TODO: We need to drop this session id auth header req and move to a token based system
	headers := make(map[string]string)
	headers["Authorization"] = authHeader

	params := make(map[string]string)
	params["session_id"] = sessionId

	hubEndpoint := "/api/v1/hub/kube"

	ret.WebsocketClient = CommonWebsocketClient.NewCommonWebsocketClient(serviceURL, hubEndpoint, params, headers)

	select {}
}
