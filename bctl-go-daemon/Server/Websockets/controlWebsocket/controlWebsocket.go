package controlWebsocket

import (
	"bytes"
	"context"
	"encoding/json"
	"log"

	"bastionzero.com/bctl/v1/Server/Websockets/controlWebsocket/controlWebsocketTypes"
	"bastionzero.com/bctl/v1/Server/Websockets/controlWebsocket/plugins/alivecheck"
	"bastionzero.com/bctl/v1/commonWebsocketClient"
)

// Constructor to create a new Control Websocket Client
func NewControlWebsocketClient(serviceURL string, activationToken string, orgId string, clusterName string, environmentId string) *controlWebsocketTypes.ControlWebsocket {
	ret := controlWebsocketTypes.ControlWebsocket{}

	// Create our headers and params, headers are empty
	headers := make(map[string]string)

	// Make and add our params
	params := make(map[string]string)
	params["activation_token"] = activationToken
	params["org_id"] = orgId
	params["cluster_name"] = clusterName
	params["environment_id"] = environmentId

	hubEndpoint := "/api/v1/hub/kube-control"

	log.Printf("serviceURL: %v, \nhubEndpoint: %V, \nparams: %V, \nheaders: %v", serviceURL, hubEndpoint, params, headers)

	// Create our response channels
	ret.ProvisionWebsocketChan = make(chan controlWebsocketTypes.ProvisionNewWebsocketMessage)
	ret.AliveCheckChan = make(chan controlWebsocketTypes.AliveCheckToClusterFromBastionSignalRMessage)

	ret.WebsocketClient = commonWebsocketClient.NewCommonWebsocketClient(serviceURL, hubEndpoint, params, headers)

	// Make our cancel context, unused for now
	ctx, _ := context.WithCancel(context.Background())

	// Set up our handler to deal with incoming messages
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case message := <-ret.WebsocketClient.WebsocketMessageChan:
				if bytes.Contains(message, []byte("\"target\":\"ProvisionNewWebsocket\"")) {
					log.Printf("Handling incoming ProvisionNewWebsocket message")

					// Unmarshall the message
					provisionWebsocketSignalRMessage := new(controlWebsocketTypes.ProvisionNewWebsocketSignalRMessage)
					err := json.Unmarshal(message, provisionWebsocketSignalRMessage)
					if err != nil {
						log.Printf("Error un-marshalling ProvisionNewWebsocket: %s", err)
						break
					}

					// Alert on our ProvisionWebsocketChan
					ret.ProvisionWebsocketChan <- provisionWebsocketSignalRMessage.Arguments[0]
				} else if bytes.Contains(message, []byte("\"target\":\"AliveCheckToClusterFromBastion\"")) {
					log.Printf("Handling incoming AliveCheckToClusterFromBastion message")
					aliveCheckToClusterFromBastionSignalRMessage := new(controlWebsocketTypes.AliveCheckToClusterFromBastionSignalRMessage)

					err := json.Unmarshal(message, aliveCheckToClusterFromBastionSignalRMessage)
					if err != nil {
						log.Printf("Error un-marshalling AliveCheckToClusterFromBastion: %s", err)
						break
					}
					ret.AliveCheckChan <- *aliveCheckToClusterFromBastionSignalRMessage
				} else {
					log.Printf("Unhandled incoming message: %s", string(message))
				}
				break
			}
		}
	}()

	// Set up our plugins
	go func() {
		for {
			message := controlWebsocketTypes.AliveCheckToClusterFromBastionSignalRMessage{}
			select {
			case <-ctx.Done():
				return
			case message = <-ret.AliveCheckChan:
				go alivecheck.AliveCheck(message, &ret)
				break
			}
		}
	}()

	return &ret
}
