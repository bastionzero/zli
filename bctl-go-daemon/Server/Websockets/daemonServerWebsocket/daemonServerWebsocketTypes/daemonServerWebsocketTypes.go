package daemonServerWebsocketTypes

import (
	"encoding/json"
	"log"
	"sync"

	"bastionzero.com/bctl/v1/commonWebsocketClient"

	"github.com/gorilla/websocket"
)

type RequestBastionToClusterSignalRMessage struct {
	Target    string                    `json:"target"`
	Arguments []RequestBastionToCluster `json:"arguments"`
	Type      int                       `json:"type"`
}
type RequestBastionToCluster struct {
	Endpoint          string            `json:"endpoint"`
	Headers           map[string]string `json:"headers"`
	Method            string            `json:"method"`
	Body              []byte            `json:"body"`
	RequestIdentifier int               `json:"requestIdentifier"`
	Role              string            `json"role"`
}

type ResponseClusterToBastionSignalRMessage struct {
	Target    string                     `json:"target"`
	Arguments []ResponseClusterToBastion `json:"arguments"`
	Type      int                        `json:"type"`
}
type ResponseClusterToBastion struct {
	StatusCode        int               `json:"statusCode"`
	Content           []byte            `json:"content"`
	RequestIdentifier int               `json:"requestIdentifier"`
	Headers           map[string]string `json:"headers"`
}

type StartExecToClusterFromBastionSignalRMessage struct {
	Target    string                                 `json:"target"`
	Arguments []StartExecToClusterFromBastionMessage `json:"arguments"`
	Type      int                                    `json:"type"`
}
type StartExecToClusterFromBastionMessage struct {
	Command           []string `json:"command"`
	Endpoint          string   `json:"endpoint"`
	RequestIdentifier int      `json:"requestIdentifier"`
	Role              string   `json"role"`
}

type StdoutToBastionFromClusterSignalRMessage struct {
	Target    string                              `json:"target"`
	Arguments []StdoutToBastionFromClusterMessage `json:"arguments"`
	Type      int                                 `json:"type"`
}
type StdoutToBastionFromClusterMessage struct {
	Stdout            []byte `json:"stdout"`
	RequestIdentifier int    `json:"requestIdentifier"`
}

type StderrToBastionFromClusterSignalRMessage struct {
	Target    string                              `json:"target"`
	Arguments []StderrToBastionFromClusterMessage `json:"arguments"`
	Type      int                                 `json:"type"`
}
type StderrToBastionFromClusterMessage struct {
	Stderr            []byte `json:"stderr"`
	RequestIdentifier int    `json:"requestIdentifier"`
}

type SendStdoutToDaemonSignalRMessage struct {
	Target    string                      `json:"target"`
	Arguments []SendStdoutToDaemonMessage `json:"arguments"`
	Type      int                         `json:"type"`
}
type SendStdoutToDaemonMessage struct {
	Stdout            []byte `json:"stdout"`
	RequestIdentifier int    `json:"requestIdentifier"`
}

type StdinToClusterFromBastionSignalRMessage struct {
	Target    string                             `json:"target"`
	Arguments []StdinToClusterFromBastionMessage `json:"arguments"`
	Type      int                                `json:"type"`
}
type StdinToClusterFromBastionMessage struct {
	Stdin             []byte `json:"stdin"`
	End               bool   `json:"end"`
	RequestIdentifier int    `json:"requestIdentifier"`
}

type ResizeTerminalToClusterFromBastionSignalRMessage struct {
	Target    string                                      `json:"target"`
	Arguments []ResizeTerminalToClusterFromBastionMessage `json:"arguments"`
	Type      int                                         `json:"type"`
}
type ResizeTerminalToClusterFromBastionMessage struct {
	Width             uint16 `json:"width"`
	Height            uint16 `json:"height"`
	RequestIdentifier int    `json:"requestIdentifier"`
}

type CloseConnectionToClusterFromBastionSignalRMessage struct {
	Target    string                                       `json:"target"`
	Arguments []CloseConnectionToClusterFromBastionMessage `json:"arguments"`
	Type      int                                          `json:"type"`
}
type CloseConnectionToClusterFromBastionMessage struct {
}

// Daemon Websock
type DaemonServerWebsocket struct {
	WebsocketClient *commonWebsocketClient.WebsocketClient

	// These are all the    types of channels we have available
	// Basic REST Call related
	RequestForServerChan     chan RequestBastionToCluster
	RequestForServerChanLock sync.Mutex

	// Logs Related
	RequestLogForServerChan     chan RequestBastionToCluster
	RequestLogForServerChanLock sync.Mutex

	// Exec Related
	RequestForStartExecChan     chan StartExecToClusterFromBastionSignalRMessage
	RequestForStartExecChanLock sync.Mutex
	ExecStdoutChan              chan SendStdoutToDaemonSignalRMessage
	ExecStdoutChanLock          sync.Mutex
	ExecStdinChannel            chan StdinToClusterFromBastionSignalRMessage
	ExecStdinChannelLock        sync.Mutex
	ExecResizeChannel           chan ResizeTerminalToClusterFromBastionSignalRMessage
	ExecResizeChannelLock       sync.Mutex

	SocketLock sync.Mutex // Ref: https://github.com/gorilla/websocket/issues/119#issuecomment-198710015
}

func (client *DaemonServerWebsocket) AlertOnRequestForServerChan(requestToClusterFromBastionMessage RequestBastionToCluster) {
	// Lock our mutex and setup the unlock
	client.RequestForServerChanLock.Lock()
	defer client.RequestForServerChanLock.Unlock()
	client.RequestForServerChan <- requestToClusterFromBastionMessage
}

func (client *DaemonServerWebsocket) AlertOnRequestLogForServerChan(requestLogBastionToCluster RequestBastionToCluster) {
	// Lock our mutex and setup the unlock
	client.RequestLogForServerChanLock.Lock()
	defer client.RequestLogForServerChanLock.Unlock()
	client.RequestLogForServerChan <- requestLogBastionToCluster
}

func (client *DaemonServerWebsocket) AlertOnRequestForStartExecChan(startExecToClusterFromBastionSignalRMessage StartExecToClusterFromBastionSignalRMessage) {
	// Lock our mutex and setup the unlock
	client.RequestForStartExecChanLock.Lock()
	defer client.RequestForStartExecChanLock.Unlock()
	client.RequestForStartExecChan <- startExecToClusterFromBastionSignalRMessage
}

func (client *DaemonServerWebsocket) AlertOnExecStdoutChan(sendStdoutToDaemonSignalRMessage SendStdoutToDaemonSignalRMessage) {
	// Lock our mutex and setup the unlock
	client.ExecStdoutChanLock.Lock()
	defer client.ExecStdoutChanLock.Unlock()
	client.ExecStdoutChan <- sendStdoutToDaemonSignalRMessage
}

func (client *DaemonServerWebsocket) AlertOnExecStdinChan(stdinToClusterFromBastionSignalRMessage StdinToClusterFromBastionSignalRMessage) {
	// Lock our mutex and setup the unlock
	client.ExecStdinChannelLock.Lock()
	defer client.ExecStdinChannelLock.Unlock()
	client.ExecStdinChannel <- stdinToClusterFromBastionSignalRMessage
}

func (client *DaemonServerWebsocket) AlertOnExecResizeChan(resizeTerminalToClusterFromBastionSingalRMessage ResizeTerminalToClusterFromBastionSignalRMessage) {
	// Lock our mutex and setup the unlock
	client.ExecResizeChannelLock.Lock()
	defer client.ExecResizeChannelLock.Unlock()
	client.ExecResizeChannel <- resizeTerminalToClusterFromBastionSingalRMessage
}

func (client *DaemonServerWebsocket) SendResponseClusterToBastion(responseToBastionFromClusterMessage ResponseClusterToBastion) error {
	// Lock our mutex and setup the unlock
	client.SocketLock.Lock()
	defer client.SocketLock.Unlock()

	log.Printf("Sending Response to To Bastion")
	// Create the object, add relevent information
	toSend := new(ResponseClusterToBastionSignalRMessage)
	toSend.Target = "ResponseToBastionFromCluster"
	toSend.Arguments = []ResponseClusterToBastion{responseToBastionFromClusterMessage}

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

func (client *DaemonServerWebsocket) SendResponseLogClusterToBastion(responseLogClusterToBastion ResponseClusterToBastion) error {
	// Lock our mutex and setup the unlock
	client.SocketLock.Lock()
	defer client.SocketLock.Unlock()

	log.Printf("Sending Log Response to Bastion")
	// Create the object, add relevent information
	toSend := new(ResponseClusterToBastionSignalRMessage)
	toSend.Target = "ResponseLogToBastionFromCluster"
	toSend.Arguments = []ResponseClusterToBastion{responseLogClusterToBastion}

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

	return nil
}

func (client *DaemonServerWebsocket) SendStdoutToBastionFromClusterMessage(stdoutToBastionFromClusterMessage StdoutToBastionFromClusterMessage) error {
	// Lock our mutex and setup the unlock
	client.SocketLock.Lock()
	defer client.SocketLock.Unlock()

	log.Printf("Sending Stdout to Bastion")
	// Create the object, add relevent information
	toSend := new(StdoutToBastionFromClusterSignalRMessage)
	toSend.Target = "StdoutToBastionFromCluster"
	toSend.Arguments = []StdoutToBastionFromClusterMessage{stdoutToBastionFromClusterMessage}

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

func (client *DaemonServerWebsocket) SendStderrToBastionFromClusterMessage(stderrToBastionFromClusterMessage StderrToBastionFromClusterMessage) error {
	// Lock our mutex and setup the unlock
	client.SocketLock.Lock()
	defer client.SocketLock.Unlock()

	log.Printf("Sending Stderr to Bastion")
	// Create the object, add relevent information
	toSend := new(StderrToBastionFromClusterSignalRMessage)
	toSend.Target = "StderrToBastionFromCluster"
	toSend.Arguments = []StderrToBastionFromClusterMessage{stderrToBastionFromClusterMessage}

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
