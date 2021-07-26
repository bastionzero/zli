package daemonWebsocket

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"sync"

	"bastionzero.com/bctl/v1/Daemon/daemonWebsocket/daemonWebsocketTypes"
	"bastionzero.com/bctl/v1/commonWebsocketClient"

	"github.com/gorilla/websocket"
)

type DaemonWebsocket struct {
	WebsocketClient *commonWebsocketClient.WebsocketClient

	// Flag to indicate once we've provisioned a websocket
	IsReady bool

	// These are all the    	types of channels we have available
	ResponseToDaemonChan        chan daemonWebsocketTypes.ResponseBastionToDaemon
	ResponseToDaemonChanLock    sync.Mutex
	ResponseLogToDaemonChan     chan daemonWebsocketTypes.ResponseBastionToDaemon
	ResponseLogToDaemonChanLock sync.Mutex
	ExecStdoutChan              chan daemonWebsocketTypes.StdoutToDaemonFromBastionSignalRMessage
	ExecStdoutChanLock          sync.Mutex
	ExecStderrChan              chan daemonWebsocketTypes.StderrToDaemonFromBastionSignalRMessage
	ExecStderrChanLock          sync.Mutex

	SocketLock sync.Mutex // Ref: https://github.com/gorilla/websocket/issues/119#issuecomment-198710015
}

// Constructor to create a new Control Websocket Client
func NewDaemonWebsocketClient(sessionId string, authHeader string, serviceURL string, assumeRole string, assumeCluster string, environmentId string) *DaemonWebsocket {
	ret := DaemonWebsocket{}

	// Set us to not ready
	ret.IsReady = false

	// Create our headers and params
	headers := make(map[string]string)
	headers["Authorization"] = authHeader

	// Add our token to our params
	params := make(map[string]string)
	params["session_id"] = sessionId
	params["assume_role"] = assumeRole
	params["assume_cluster"] = assumeCluster
	params["environment_id"] = environmentId

	hubEndpoint := "/api/v1/hub/kube"

	// Add our response channels
	ret.ResponseToDaemonChan = make(chan daemonWebsocketTypes.ResponseBastionToDaemon)
	ret.ResponseLogToDaemonChan = make(chan daemonWebsocketTypes.ResponseBastionToDaemon)
	ret.ExecStdoutChan = make(chan daemonWebsocketTypes.StdoutToDaemonFromBastionSignalRMessage)
	ret.ExecStderrChan = make(chan daemonWebsocketTypes.StderrToDaemonFromBastionSignalRMessage)

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
				if bytes.Contains(message, []byte("\"target\":\"ReadyToClientFromBastion\"")) {
					log.Printf("Handling incoming ReadyToClient message")
					readyToClientFromBastionSignalRMessage := new(daemonWebsocketTypes.ReadyToClientFromBastionSignalRMessage)
					err := json.Unmarshal(message, readyToClientFromBastionSignalRMessage)
					if err != nil {
						log.Printf("Error un-marshalling ReadyToClientFromBastion: %s", err)
						break
					}
					if readyToClientFromBastionSignalRMessage.Arguments[0].Ready {
						log.Printf("Server is ready!")
						ret.IsReady = true
					} else {
						log.Printf("Server is still not ready")
					}
				} else if bytes.Contains(message, []byte("\"target\":\"ResponseToDaemonFromBastion\"")) {
					log.Printf("Handling incoming ResponseToDaemonFromBastion message")
					responseToDaemonFromBastionSignalRMessage := new(daemonWebsocketTypes.ResponseBastionToDaemonSignalRMessage)
					err := json.Unmarshal(message, responseToDaemonFromBastionSignalRMessage)
					if err != nil {
						log.Printf("Error un-marshalling ResponseToDaemonFromBastion: %s", err)
						break
					}
					// Broadcase this response to our DataToClientChan
					ret.AlertOnResponseToDaemonChan(responseToDaemonFromBastionSignalRMessage.Arguments[0])
				} else if bytes.Contains(message, []byte("\"target\":\"ResponseLogToDaemonFromBastion\"")) {
					log.Printf("Handling incoming ResponseLogToDaemonFromBastion message")
					responseLogToDaemonFromBastionSignalRMessage := new(daemonWebsocketTypes.ResponseBastionToDaemonSignalRMessage)
					err := json.Unmarshal(message, responseLogToDaemonFromBastionSignalRMessage)
					if err != nil {
						log.Printf("Error un-marshalling ResponseLogToDaemonFromBastion: %s", err)
						break
					}
					// Broadcase this response to our DataToClientChan
					ret.AlertOnResponseLogToDaemonChan(responseLogToDaemonFromBastionSignalRMessage.Arguments[0])
				} else if bytes.Contains(message, []byte("\"target\":\"StdoutToDaemonFromBastion\"")) {
					log.Printf("Handling incoming StdoutToDaemonFromBastion message")
					stdoutToDaemonFromBastionSignalRMessage := new(daemonWebsocketTypes.StdoutToDaemonFromBastionSignalRMessage)

					err := json.Unmarshal(message, stdoutToDaemonFromBastionSignalRMessage)
					if err != nil {
						log.Printf("Error un-marshalling StdoutToDaemonFromBastion: %s", err)
						break
					}
					// Broadcase this response to our RequestForStartExecChan
					ret.AlertOnExecStdoutChan(*stdoutToDaemonFromBastionSignalRMessage)
				} else if bytes.Contains(message, []byte("\"target\":\"StderrToDaemonFromBastion\"")) {
					log.Printf("Handling incoming StderrToDaemonFromBastion message")
					stderrToDaemonFromBastionSignalRMessage := new(daemonWebsocketTypes.StderrToDaemonFromBastionSignalRMessage)

					err := json.Unmarshal(message, stderrToDaemonFromBastionSignalRMessage)
					if err != nil {
						log.Printf("Error un-marshalling StderrToDaemonFromBastion: %s", err)
						break
					}
					// Broadcase this response to our RequestForStartExecChan
					ret.AlertOnExecStderrChan(*stderrToDaemonFromBastionSignalRMessage)
				} else {
					log.Printf("Unhandled incoming message: %s", string(message))
				}
				break
			}
		}
	}()

	return &ret
}

func (client *DaemonWebsocket) AlertOnExecStderrChan(stderrToDaemonFromBastionSignalRMessage daemonWebsocketTypes.StderrToDaemonFromBastionSignalRMessage) {
	// Lock our mutex and setup the unlock
	client.ExecStderrChanLock.Lock()
	defer client.ExecStderrChanLock.Unlock()
	client.ExecStderrChan <- stderrToDaemonFromBastionSignalRMessage
}

func (client *DaemonWebsocket) AlertOnExecStdoutChan(stdoutToDaemonFromBastionSignalRMessage daemonWebsocketTypes.StdoutToDaemonFromBastionSignalRMessage) {
	// Lock our mutex and setup the unlock
	client.ExecStdoutChanLock.Lock()
	defer client.ExecStdoutChanLock.Unlock()
	client.ExecStdoutChan <- stdoutToDaemonFromBastionSignalRMessage
}

func (client *DaemonWebsocket) AlertOnResponseToDaemonChan(responseToDaemonFromBastionMessage daemonWebsocketTypes.ResponseBastionToDaemon) {
	// Lock our mutex and setup the unlock
	client.ResponseToDaemonChanLock.Lock()
	defer client.ResponseToDaemonChanLock.Unlock()
	client.ResponseToDaemonChan <- responseToDaemonFromBastionMessage
}

func (client *DaemonWebsocket) AlertOnResponseLogToDaemonChan(responseLogBastionToDaemon daemonWebsocketTypes.ResponseBastionToDaemon) {
	// Lock our mutex and setup the unlock
	client.ResponseLogToDaemonChanLock.Lock()
	defer client.ResponseLogToDaemonChanLock.Unlock()
	client.ResponseLogToDaemonChan <- responseLogBastionToDaemon
}

// Function to send data Bastion from a RequestToBastionFromDaemonMessage object
func (client *DaemonWebsocket) SendRequestDaemonToBastion(dataFromClientMessage daemonWebsocketTypes.RequestDaemonToBastion) error {
	if client.IsReady {

		// Lock our mutex and setup the unlock
		client.SocketLock.Lock()
		defer client.SocketLock.Unlock()

		log.Printf("Sending data to Bastion")

		// Create the object, add relevent information
		toSend := new(daemonWebsocketTypes.RequestDaemonToBastionSignalRMessage)
		toSend.Target = "RequestToBastionFromDaemon"
		toSend.Arguments = []daemonWebsocketTypes.RequestDaemonToBastion{dataFromClientMessage}

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
	// TODO: Return error
	return nil
}

// Function to send data to Bastion from a RequestLogDaemonToBastion object
func (client *DaemonWebsocket) SendRequestLogDaemonToBastion(dataFromClientMessage daemonWebsocketTypes.RequestDaemonToBastion) error {
	if client.IsReady {

		// Lock our mutex and setup the unlock
		client.SocketLock.Lock()
		defer client.SocketLock.Unlock()

		log.Printf("Sending log data to Bastion")

		// Create the object, add relevent information
		toSend := new(daemonWebsocketTypes.RequestDaemonToBastionSignalRMessage)
		toSend.Target = "RequestLogToBastionFromDaemon"
		toSend.Arguments = []daemonWebsocketTypes.RequestDaemonToBastion{dataFromClientMessage}

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
	// TODO: Return error
	return nil
}

// Helper function to generate a random unique identifier
func (c *DaemonWebsocket) GenerateUniqueIdentifier() int {
	for {
		i := rand.Intn(10000)
		return i
		// TODO: Implement a unique check
		// if !u.generated[i] {
		// 	u.generated[i] = true
		// 	return i
		// }
	}
}

// Function to send Exec stdin to Bastion
func (client *DaemonWebsocket) SendStdinDaemonToBastion(stdinToBastionFromDaemonMessage daemonWebsocketTypes.StdinToBastionFromDaemonMessage) error {
	if client.IsReady {
		// Lock our mutex and setup the unlock
		client.SocketLock.Lock()
		defer client.SocketLock.Unlock()

		log.Printf("Sending stdin to Cluster")
		// Create the object, add relevent information
		toSend := new(daemonWebsocketTypes.StdinToBastionFromDaemonSignalRMessage)
		toSend.Target = "StdinToBastionFromDaemon"
		toSend.Arguments = []daemonWebsocketTypes.StdinToBastionFromDaemonMessage{stdinToBastionFromDaemonMessage}

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
	// TODO: Return error
	return nil
}

// Function to send Exec resize events to Bastion
func (client *DaemonWebsocket) SendResizeDaemonToBastion(resizeTerminalToBastionFromDaemonMessage daemonWebsocketTypes.ResizeTerminalToBastionFromDaemonMessage) error {
	if client.IsReady {
		// Lock our mutex and setup the unlock
		client.SocketLock.Lock()
		defer client.SocketLock.Unlock()

		log.Printf("Sending stdin to Cluster")
		// Create the object, add relevent information
		toSend := new(daemonWebsocketTypes.ResizeTerminalToBastionFromDaemonSignalRMessage)
		toSend.Target = "ResizeTerminalToBastionFromDaemon"
		toSend.Arguments = []daemonWebsocketTypes.ResizeTerminalToBastionFromDaemonMessage{resizeTerminalToBastionFromDaemonMessage}

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
	// TODO: Return error
	return nil
}

func (client *DaemonWebsocket) SendStartExecDaemonToBastion(startExecToBastionMessage daemonWebsocketTypes.StartExecToBastionFromDaemonMessage) error {
	if client.IsReady {
		// Lock our mutex and setup the unlock
		client.SocketLock.Lock()
		defer client.SocketLock.Unlock()

		log.Printf("Sending data to Cluster")
		// Create the object, add relevent information
		toSend := new(daemonWebsocketTypes.StartExecToBastionFromDaemonSignalRMessage)
		toSend.Target = "StartExecToBastionFromDaemon"
		toSend.Arguments = []daemonWebsocketTypes.StartExecToBastionFromDaemonMessage{startExecToBastionMessage}

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
	// TODO: Return error
	return nil
}
