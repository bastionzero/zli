package controlWebsocket

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"sync"

	"bastionzero.com/bctl/v1/Server/Websockets/controlWebsocket/controlWebsocketTypes"
	"bastionzero.com/bctl/v1/commonWebsocketClient"

	"github.com/gorilla/websocket"
)

type ControlWebsocket struct {
	WebsocketClient *commonWebsocketClient.WebsocketClient

	// These are all the    types of channels we have available
	ProvisionWebsocketChan chan controlWebsocketTypes.ProvisionNewWebsocketMessage

	SocketLock sync.Mutex // Ref: https://github.com/gorilla/websocket/issues/119#issuecomment-198710015
}

// Constructor to create a new Control Websocket Client
func NewControlWebsocketClient(serviceURL string, activationToken string, orgId string, clusterName string, environmentId string) *ControlWebsocket {
	ret := ControlWebsocket{}

	// Create our headers and params, headers are empty
	headers := make(map[string]string)

	// Make and add our params
	params := make(map[string]string)
	params["activation_token"] = activationToken
	params["org_id"] = orgId
	params["cluster_name"] = clusterName
	params["environment_id"] = environmentId

	hubEndpoint := "/api/v1/hub/kube-control"

	// Create our response channels
	ret.ProvisionWebsocketChan = make(chan controlWebsocketTypes.ProvisionNewWebsocketMessage)

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
					// Let the Bastion know we are alive!
					aliveCheckToBastionFromClusterMessage := new(controlWebsocketTypes.AliveCheckToBastionFromClusterMessage)
					aliveCheckToBastionFromClusterMessage.Alive = true
					ret.SendAliveCheckToBastionFromClusterMessage(*aliveCheckToBastionFromClusterMessage)
				} else {
					log.Printf("Unhandled incoming message: %s", string(message))
				}
				break
			}
		}
	}()

	return &ret
}

func (client *ControlWebsocket) SendAliveCheckToBastionFromClusterMessage(aliveCheckToBastionFromClusterMessage controlWebsocketTypes.AliveCheckToBastionFromClusterMessage) error {
	// Lock our mutex and setup the unlock
	client.SocketLock.Lock()
	defer client.SocketLock.Unlock()

	log.Printf("Sending data to Daemon")
	// Create the object, add relevent information
	toSend := new(controlWebsocketTypes.AliveCheckToBastionFromClusterSignalRMessage)
	toSend.Target = "AliveCheckToBastionFromCluster"
	toSend.Arguments = []controlWebsocketTypes.AliveCheckToBastionFromClusterMessage{aliveCheckToBastionFromClusterMessage}

	// Add the type number from the class
	toSend.Type = 1 // Ref: https://github.com/aspnet/SignalR/blob/master/specs/HubProtocol.md#invocation-message-encoding

	// Marshal our message
	toSendMarshalled, err := json.Marshal(toSend)
	if err != nil {
		return err
	}

	// Write our message
	if err = client.WebsocketClient.Client.WriteMessage(websocket.TextMessage, append(toSendMarshalled, 0x1E)); err != nil {
		return err
	}
	// client.SignalRTypeNumber++
	return nil
}
