package DaemonWebsocket

import (
	"bytes"
	"encoding/json"
	"log"
	"math/rand"
	"sync"

	"bastionzero.com/bctl/v1/CommonWebsocketClient"
	"github.com/gorilla/websocket"
)

type DaemonWebsocket struct {
	WebsocketClient *CommonWebsocketClient.WebsocketClient

	// Flag to indicate once we've provisioned a websocket
	IsReady bool

	// These are all the    types of channels we have available
	ResponseToDaemonChan     chan ResponseToDaemonFromBastionMessage
	ResponseToDaemonChanLock sync.Mutex
	ExecStdoutChan           chan StdoutToDaemonFromBastionSignalRMessage
	ExecStdoutChanLock       sync.Mutex
	ExecStderrChan           chan StderrToDaemonFromBastionSignalRMessage
	ExecStderrChanLock       sync.Mutex

	SocketLock sync.Mutex // Ref: https://github.com/gorilla/websocket/issues/119#issuecomment-198710015
}

// Constructor to create a new Control Websocket Client
func NewDaemonWebsocketClient(sessionId string, authHeader string, serviceURL string, assumeRole string, assumeCluster string) *DaemonWebsocket {
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
	params["assume_cluster"] = assumeCluster // TODO: This doesnt do anything right now

	hubEndpoint := "/api/v1/hub/kube"

	// Add our response channels
	ret.ResponseToDaemonChan = make(chan ResponseToDaemonFromBastionMessage)
	ret.ExecStdoutChan = make(chan StdoutToDaemonFromBastionSignalRMessage)
	ret.ExecStderrChan = make(chan StderrToDaemonFromBastionSignalRMessage)

	ret.WebsocketClient = CommonWebsocketClient.NewCommonWebsocketClient(serviceURL, hubEndpoint, params, headers)

	// Set up our handler to deal with incoming messages
	go func() {
		for {
			message := <-ret.WebsocketClient.WebsocketMessageChan
			if bytes.Contains(message, []byte("\"target\":\"ReadyToClientFromBastion\"")) {
				log.Printf("Handling incoming ReadyToClient message")
				readyToClientFromBastionSignalRMessage := new(ReadyToClientFromBastionSignalRMessage)
				err := json.Unmarshal(message, readyToClientFromBastionSignalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling ReadyToClientFromBastion: %s", err)
					return
				}
				if readyToClientFromBastionSignalRMessage.Arguments[0].Ready == true {
					log.Printf("Server is ready!")
					ret.IsReady = true
				} else {
					log.Printf("Server is still not ready")
				}
			} else if bytes.Contains(message, []byte("\"target\":\"ResponseToDaemonFromBastion\"")) {
				log.Printf("Handling incoming ResponseToDaemonFromBastion message")
				responseToDaemonFromBastionSignalRMessage := new(ResponseToDaemonFromBastionSignalRMessage)
				err := json.Unmarshal(message, responseToDaemonFromBastionSignalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling ResponseToDaemonFromBastion: %s", err)
					return
				}
				// Broadcase this response to our DataToClientChan
				ret.AlertOnResponseToDaemonChan(responseToDaemonFromBastionSignalRMessage.Arguments[0])
			} else if bytes.Contains(message, []byte("\"target\":\"StdoutToDaemonFromBastion\"")) {
				log.Printf("Handling incoming StdoutToDaemonFromBastion message")
				stdoutToDaemonFromBastionSignalRMessage := new(StdoutToDaemonFromBastionSignalRMessage)

				err := json.Unmarshal(message, stdoutToDaemonFromBastionSignalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling StdoutToDaemonFromBastion: %s", err)
					return
				}
				// Broadcase this response to our RequestForStartExecChan
				ret.AlertOnExecStdoutChan(*stdoutToDaemonFromBastionSignalRMessage)
			} else if bytes.Contains(message, []byte("\"target\":\"StderrToDaemonFromBastion\"")) {
				log.Printf("Handling incoming StderrToDaemonFromBastion message")
				stderrToDaemonFromBastionSignalRMessage := new(StderrToDaemonFromBastionSignalRMessage)

				err := json.Unmarshal(message, stderrToDaemonFromBastionSignalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling StderrToDaemonFromBastion: %s", err)
					return
				}
				// Broadcase this response to our RequestForStartExecChan
				ret.AlertOnExecStderrChan(*stderrToDaemonFromBastionSignalRMessage)
			} else {
				log.Printf("Unhandled incoming message: %s", string(message))
			}
		}
	}()

	return &ret
}

func (client *DaemonWebsocket) AlertOnExecStderrChan(stderrToDaemonFromBastionSignalRMessage StderrToDaemonFromBastionSignalRMessage) {
	// Lock our mutex and setup the unlock
	client.ExecStderrChanLock.Lock()
	defer client.ExecStderrChanLock.Unlock()
	client.ExecStderrChan <- stderrToDaemonFromBastionSignalRMessage
}

func (client *DaemonWebsocket) AlertOnExecStdoutChan(stdoutToDaemonFromBastionSignalRMessage StdoutToDaemonFromBastionSignalRMessage) {
	// Lock our mutex and setup the unlock
	client.ExecStdoutChanLock.Lock()
	defer client.ExecStdoutChanLock.Unlock()
	client.ExecStdoutChan <- stdoutToDaemonFromBastionSignalRMessage
}

func (client *DaemonWebsocket) AlertOnResponseToDaemonChan(responseToDaemonFromBastionMessage ResponseToDaemonFromBastionMessage) {
	// Lock our mutex and setup the unlock
	client.ResponseToDaemonChanLock.Lock()
	defer client.ResponseToDaemonChanLock.Unlock()
	client.ResponseToDaemonChan <- responseToDaemonFromBastionMessage
}

// Function to send data Bastion from a RequestToBastionFromDaemonMessage object
func (client *DaemonWebsocket) SendRequestToBastionFromDaemonMessage(dataFromClientMessage RequestToBastionFromDaemonMessage) error {
	if client.IsReady {

		// Lock our mutex and setup the unlock
		client.SocketLock.Lock()
		defer client.SocketLock.Unlock()

		log.Printf("Sending data to Bastion")

		// Create the object, add relevent information
		toSend := new(RequestToBastionFromDaemonSignalRMessage)
		toSend.Target = "RequestToBastionFromDaemon"
		toSend.Arguments = []RequestToBastionFromDaemonMessage{dataFromClientMessage}

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
func (client *DaemonWebsocket) SendStdinToBastionFromDaemonMessage(stdinToBastionFromDaemonMessage StdinToBastionFromDaemonMessage) error {
	if client.IsReady {
		// Lock our mutex and setup the unlock
		client.SocketLock.Lock()
		defer client.SocketLock.Unlock()

		log.Printf("Sending stdin to Cluster")
		// Create the object, add relevent information
		toSend := new(StdinToBastionFromDaemonSignalRMessage)
		toSend.Target = "StdinToBastionFromDaemon"
		toSend.Arguments = []StdinToBastionFromDaemonMessage{stdinToBastionFromDaemonMessage}

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
func (client *DaemonWebsocket) SendResizeTerminalToBastionFromDaemonMessage(resizeTerminalToBastionFromDaemonMessage ResizeTerminalToBastionFromDaemonMessage) error {
	if client.IsReady {
		// Lock our mutex and setup the unlock
		client.SocketLock.Lock()
		defer client.SocketLock.Unlock()

		log.Printf("Sending stdin to Cluster")
		// Create the object, add relevent information
		toSend := new(ResizeTerminalToBastionFromDaemonSignalRMessage)
		toSend.Target = "ResizeTerminalToBastionFromDaemon"
		toSend.Arguments = []ResizeTerminalToBastionFromDaemonMessage{resizeTerminalToBastionFromDaemonMessage}

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

func (client *DaemonWebsocket) SendStartExecToBastionFromDaemonMessage(startExecToBastionMessage StartExecToBastionFromDaemonMessage) error {
	if client.IsReady {
		// Lock our mutex and setup the unlock
		client.SocketLock.Lock()
		defer client.SocketLock.Unlock()

		log.Printf("Sending data to Cluster")
		// Create the object, add relevent information
		toSend := new(StartExecToBastionFromDaemonSignalRMessage)
		toSend.Target = "StartExecToBastionFromDaemon"
		toSend.Arguments = []StartExecToBastionFromDaemonMessage{startExecToBastionMessage}

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
