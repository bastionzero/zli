package controlWebsocketTypes

import (
	"encoding/json"
	"log"
	"sync"

	"bastionzero.com/bctl/v1/commonWebsocketClient"

	"github.com/gorilla/websocket"
)

type ProvisionNewWebsocketSignalRMessage struct {
	Target    string                         `json:"target"`
	Arguments []ProvisionNewWebsocketMessage `json:"arguments"`
	Type      int                            `json:"type"`
}

type ProvisionNewWebsocketMessage struct {
	ConnectionId string `json:"connectionId"`
}

type AliveCheckToClusterFromBastionSignalRMessage struct {
	Target    string                                  `json:"target"`
	Arguments []AliveCheckToClusterFromBastionMessage `json:"arguments"`
	Type      int                                     `json:"type"`
}
type AliveCheckToClusterFromBastionMessage struct {
}

type AliveCheckToBastionFromClusterSignalRMessage struct {
	Target    string                                  `json:"target"`
	Arguments []AliveCheckToBastionFromClusterMessage `json:"arguments"`
	Type      int                                     `json:"type"`
}
type AliveCheckToBastionFromClusterMessage struct {
	Alive        bool     `json:"alive"`
	ClusterRoles []string `json:"clusterRoles`
}

type ControlWebsocket struct {
	WebsocketClient *commonWebsocketClient.WebsocketClient

	// These are all the    types of channels we have available
	ProvisionWebsocketChan chan ProvisionNewWebsocketMessage
	AliveCheckChan         chan AliveCheckToClusterFromBastionSignalRMessage

	SocketLock sync.Mutex // Ref: https://github.com/gorilla/websocket/issues/119#issuecomment-198710015
}

func (client *ControlWebsocket) SendAliveCheckToBastionFromClusterMessage(aliveCheckToBastionFromClusterMessage AliveCheckToBastionFromClusterMessage) error {
	// Lock our mutex and setup the unlock
	client.SocketLock.Lock()
	defer client.SocketLock.Unlock()

	log.Printf("Sending AliveCheck to Bastion")
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
