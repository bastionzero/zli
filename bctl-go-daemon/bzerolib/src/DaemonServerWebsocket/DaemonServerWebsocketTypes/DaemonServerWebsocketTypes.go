package DaemonServerWebsocketTypes

import (
	"encoding/json"
	"log"
	"sync"

	"bastionzero.com/bctl/v1/CommonWebsocketClient"

	"github.com/gorilla/websocket"
)

type RequestForServerSignalRMessage struct {
	Target    string                    `json:"target"`
	Arguments []RequestForServerMessage `json:"arguments"`
	Type      int                       `json:"type"`
}
type RequestForServerMessage struct {
	Endpoint          string            `json:"endpoint"`
	Headers           map[string]string `json:"headers"`
	Method            string            `json:"method"`
	Body              string            `json:"body"`
	RequestIdentifier int               `json:"requestIdentifier"`
}

type ResponseToDaemonSignalRMessage struct {
	Target    string                    `json:"target"`
	Arguments []ResponseToDaemonMessage `json:"arguments"`
	Type      int                       `json:"type"`
}
type ResponseToDaemonMessage struct {
	StatusCode        int               `json:"statusCode"`
	Content           string            `json:"content"`
	RequestIdentifier int               `json:"requestIdentifier"`
	Headers           map[string]string `json:"headers"`
}

type RequestForStartExecToClusterSingalRMessage struct {
	Target    string                                `json:"target"`
	Arguments []RequestForStartExecToClusterMessage `json:"arguments"`
	Type      int                                   `json:"type"`
}
type RequestForStartExecToClusterMessage struct {
	Command           []string `json:"command"`
	Endpoint          string   `json:"endpoint"`
	RequestIdentifier int      `json:"requestIdentifier"`
}

type SendStdoutToBastionSignalRMessage struct {
	Target    string                       `json:"target"`
	Arguments []SendStdoutToBastionMessage `json:"arguments"`
	Type      int                          `json:"type"`
}
type SendStdoutToBastionMessage struct {
	Stdout            string `json:"stdout"`
	RequestIdentifier int    `json:"requestIdentifier"`
}

type SendStdoutToDaemonSignalRMessage struct {
	Target    string                      `json:"target"`
	Arguments []SendStdoutToDaemonMessage `json:"arguments"`
	Type      int                         `json:"type"`
}
type SendStdoutToDaemonMessage struct {
	Stdout            string `json:"stdout"`
	RequestIdentifier int    `json:"requestIdentifier"`
}

type SendStdinToClusterSignalRMessage struct {
	Target    string                      `json:"target"`
	Arguments []SendStdinToClusterMessage `json:"arguments"`
	Type      int                         `json:"type"`
}
type SendStdinToClusterMessage struct {
	Stdin             string `json:"stdin"`
	RequestIdentifier int    `json:"requestIdentifier"`
}

// Daemon Websock
type DaemonServerWebsocket struct {
	WebsocketClient *CommonWebsocketClient.WebsocketClient

	// These are all the    types of channels we have available
	// Basic REST Call related
	RequestForServerChan chan RequestForServerMessage

	// Exec Related
	RequestForStartExecChan chan RequestForStartExecToClusterSingalRMessage
	ExecStdoutChan          chan SendStdoutToDaemonSignalRMessage
	// RequestForServerChan    chan CommonWebsocketClient.RequestForServerSignalRMessage
	// RequestForStartExecChan chan CommonWebsocketClient.RequestForStartExecToClusterSingalRMessage
	ExecStdinChannel chan SendStdinToClusterSignalRMessage

	SocketLock sync.Mutex // Ref: https://github.com/gorilla/websocket/issues/119#issuecomment-198710015
}

func (client *DaemonServerWebsocket) SendResponseToDaemonMessage(responseToDaemonMessage ResponseToDaemonMessage) error {
	// Lock our mutex and setup the unlock
	client.SocketLock.Lock()
	defer client.SocketLock.Unlock()

	log.Printf("Sending data to Daemon")
	// Create the object, add relevent information
	toSend := new(ResponseToDaemonSignalRMessage)
	toSend.Target = "ResponseToDaemon"
	toSend.Arguments = []ResponseToDaemonMessage{responseToDaemonMessage}

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

func (client *DaemonServerWebsocket) SendSendStdoutToBastionMessage(sendStdoutToBastionMessage SendStdoutToBastionMessage) error {
	// Lock our mutex and setup the unlock
	client.SocketLock.Lock()
	defer client.SocketLock.Unlock()

	log.Printf("Sending stdout to Cluster")
	// Create the object, add relevent information
	toSend := new(SendStdoutToBastionSignalRMessage)
	toSend.Target = "SendStdoutToBastionFromCluster"
	toSend.Arguments = []SendStdoutToBastionMessage{sendStdoutToBastionMessage}

	// Add the type number from the class
	toSend.Type = 1 // Ref: https://github.com/aspnet/SignalR/blob/master/specs/HubProtocol.md#invocation-message-encoding

	// Marshal our message
	toSendMarshalled, err := json.Marshal(toSend)
	if err != nil {
		return err
	}

	// Write our message
	if err = client.WebsocketClient.Client.WriteMessage(websocket.TextMessage, append(toSendMarshalled, 0x1E)); err != nil {
		log.Printf("Something went wrong :(")
		return err
	}
	// client.SignalRTypeNumber++
	return nil
}
