package ControlWebsocket

import (
	"bytes"
	"encoding/json"
	"log"
	"sync"

	"bastionzero.com/bctl/v1/CommonWebsocketClient"

	"github.com/gorilla/websocket"
)

type ControlWebsocket struct {
	WebsocketClient *CommonWebsocketClient.WebsocketClient

	// These are all the    types of channels we have available
	ProvisionWebsocketChan chan ProvisionNewWebsocketMessage

	SocketLock sync.Mutex // Ref: https://github.com/gorilla/websocket/issues/119#issuecomment-198710015
}

// Constructor to create a new Control Websocket Client
func NewControlWebsocketClient(serviceURL string, activationToken string) *ControlWebsocket {
	ret := ControlWebsocket{}

	// Create our headers and params, headers are empty
	headers := make(map[string]string)

	// Add our token to our params
	params := make(map[string]string)
	params["activation_token"] = activationToken

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
			} else if bytes.Contains(message, []byte("\"target\":\"AliveCheckToClusterFromBastion\"")) {
				log.Printf("Handling incoming AliveCheckToClusterFromBastion message")
				aliveCheckToClusterFromBastionSignalRMessage := new(AliveCheckToClusterFromBastionSignalRMessage)

				err := json.Unmarshal(message, aliveCheckToClusterFromBastionSignalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling AliveCheckToClusterFromBastion: %s", err)
					return
				}
				// Let the Bastion know we are alive!
				aliveCheckToBastionFromClusterMessage := new(AliveCheckToBastionFromClusterMessage)
				aliveCheckToBastionFromClusterMessage.Alive = true
				ret.SendAliveCheckToBastionFromClusterMessage(*aliveCheckToBastionFromClusterMessage)
			} else {
				log.Printf("Unhandled incoming message: %s", string(message))
			}
		}
	}()

	return &ret
}

func (client *ControlWebsocket) SendAliveCheckToBastionFromClusterMessage(aliveCheckToBastionFromClusterMessage AliveCheckToBastionFromClusterMessage) error {
	// Lock our mutex and setup the unlock
	client.SocketLock.Lock()
	defer client.SocketLock.Unlock()

	log.Printf("Sending data to Daemon")
	// Create the object, add relevent information
	toSend := new(AliveCheckToBastionFromClusterSignalRMessage)
	toSend.Target = "AliveCheckToBastionFromCluster"
	toSend.Arguments = []AliveCheckToBastionFromClusterMessage{aliveCheckToBastionFromClusterMessage}

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
