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
	DataToClientChan chan DataToClientMessage
	ExecStdoutChan   chan SendStdoutToDaemonFromBastionSignalRMessage

	// RequestForServerChan    chan CommonWebsocketClient.RequestForServerSignalRMessage
	// RequestForStartExecChan chan CommonWebsocketClient.RequestForStartExecToClusterSingalRMessage
	// ExecStdinChannel        chan CommonWebsocketClient.SendStdinToClusterSignalRMessage

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
	ret.DataToClientChan = make(chan DataToClientMessage)
	ret.ExecStdoutChan = make(chan SendStdoutToDaemonFromBastionSignalRMessage)

	ret.WebsocketClient = CommonWebsocketClient.NewCommonWebsocketClient(serviceURL, hubEndpoint, params, headers)

	// Set up our handler to deal with incoming messages
	go func() {
		for {
			message := <-ret.WebsocketClient.WebsocketMessageChan
			if bytes.Contains(message, []byte("\"target\":\"ReadyToClient\"")) {
				log.Printf("Handling incoming ReadyToClient message")
				readyFromServerSignalRMessage := new(ReadyFromServerSignalRMessage)
				err := json.Unmarshal(message, readyFromServerSignalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling ReadyFromServerSignalRMessage: %s", err)
					return
				}
				if readyFromServerSignalRMessage.Arguments[0].Ready == true {
					log.Printf("Server is ready!")
					ret.IsReady = true
				} else {
					log.Printf("Server is still not ready")
				}
			} else if bytes.Contains(message, []byte("\"target\":\"DataToClient\"")) {
				log.Printf("Handling incoming DataToClient message")
				dataToClientSignalRMessage := new(DataToClientSignalRMessage)
				err := json.Unmarshal(message, dataToClientSignalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling DataToClientSignalRMessage: %s", err)
					return
				}

				// Broadcase this response to our DataToClientChan
				ret.DataToClientChan <- dataToClientSignalRMessage.Arguments[0]
			} else if bytes.Contains(message, []byte("\"target\":\"SendStdoutToDaemonFromBastion\"")) {
				log.Printf("Handling incoming SendStdoutToDaemonFromBastion message")
				sendStdoutToDaemonFromBastionSignalRMessage := new(SendStdoutToDaemonFromBastionSignalRMessage)

				err := json.Unmarshal(message, sendStdoutToDaemonFromBastionSignalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling SendStdoutToDaemonFromBastion: %s", err)
					return
				}
				// Broadcase this response to our RequestForStartExecChan
				ret.ExecStdoutChan <- *sendStdoutToDaemonFromBastionSignalRMessage
			} else {
				log.Printf("Unhandled incoming message: %s", string(message))
			}
		}
	}()

	return &ret
}

// Function to send data Bastion from a DataFromClientMessage object
func (client *DaemonWebsocket) SendDataFromClientMessage(dataFromClientMessage DataFromClientMessage) error {
	if client.IsReady {

		// Lock our mutex and setup the unlock
		client.SocketLock.Lock()
		defer client.SocketLock.Unlock()

		log.Printf("Sending data to Bastion")

		// Create the object, add relevent information
		toSend := new(DataFromClientSignalRMessage)
		toSend.Target = "DataFromClient"
		toSend.Arguments = []DataFromClientMessage{dataFromClientMessage}

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
func (client *DaemonWebsocket) SendSendStdinToBastionMessage(sendStdinToBastionMessage SendStdinToBastionMessage) error {
	if client.IsReady {
		// Lock our mutex and setup the unlock
		client.SocketLock.Lock()
		defer client.SocketLock.Unlock()

		log.Printf("Sending stdin to Cluster")
		// Create the object, add relevent information
		toSend := new(SendStdinToBastionSignalRMessage)
		toSend.Target = "SendStdinToBastion"
		toSend.Arguments = []SendStdinToBastionMessage{sendStdinToBastionMessage}

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

func (client *DaemonWebsocket) SendStartExecToBastionMessage(startExecToBastionMessage StartExecToBastionMessage) error {
	if client.IsReady {
		// Lock our mutex and setup the unlock
		client.SocketLock.Lock()
		defer client.SocketLock.Unlock()

		log.Printf("Sending data to Cluster")
		// Create the object, add relevent information
		toSend := new(StartExecToBastionSignalRMessage)
		toSend.Target = "StartExecToBastion"
		toSend.Arguments = []StartExecToBastionMessage{startExecToBastionMessage}

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
