package ControlWebsocket

import (
	"bytes"
	"encoding/json"
	"log"

	"bastionzero.com/bctl/v1/CommonWebsocketClient"
)

type ControlWebsocket struct {
	WebsocketClient *CommonWebsocketClient.WebsocketClient

	// These are all the    types of channels we have available
	ProvisionWebsocketChan chan ProvisionNewWebsocketMessage
	// RequestForServerChan    chan CommonWebsocketClient.RequestForServerSignalRMessage
	// RequestForStartExecChan chan CommonWebsocketClient.RequestForStartExecToClusterSingalRMessage
	// ExecStdinChannel        chan CommonWebsocketClient.SendStdinToClusterSignalRMessage
}

// Constructor to create a new Control Websocket Client
func NewControlWebsocketClient(serviceURL string) *ControlWebsocket {
	ret := ControlWebsocket{}

	// Create our headers and params, headers are empty
	// TODO: We need to drop this session id auth header req and move to a token based system
	headers := make(map[string]string)

	// Add our token to our params
	params := make(map[string]string)
	params["token"] = "12345"

	hubEndpoint := "/api/v1/hub/kube-control"

	// Create our response channels
	ret.ProvisionWebsocketChan = make(chan ProvisionNewWebsocketMessage)

	ret.WebsocketClient = CommonWebsocketClient.NewCommonWebsocketClient(serviceURL, hubEndpoint, params, headers)

	// Set up our handler to deal with incoming messages
	go func() {
		for {
			message := <-ret.WebsocketClient.WebsocketMessageChan
			if bytes.Contains(message, []byte("\"target\":\"ProvisionNewWebsocket\"")) {
				log.Printf("Handling incoming ProvisionNewWebsocket message")

				// Unmarshall the message
				provisionWebsocketSignalRMessage := new(ProvisionNewWebsocketSignalRMessage)
				err := json.Unmarshal(message, provisionWebsocketSignalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling ProvisionNewWebsocket: %s", err)
					return
				}

				// Alert on our ProvisionWebsocketChan
				ret.ProvisionWebsocketChan <- provisionWebsocketSignalRMessage.Arguments[0]
			} else {
				log.Printf("Unhandled incoming message: %s", string(message))
			}
		}
	}()

	return &ret
}
