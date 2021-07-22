package DaemonServerWebsocket

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"bastionzero.com/bctl/v1/CommonWebsocketClient"
	"bastionzero.com/bctl/v1/Server/src/DaemonServerWebsocket/DaemonServerWebsocketTypes"
	"bastionzero.com/bctl/v1/Server/src/DaemonServerWebsocket/Handlers/HandleExec"
	"bastionzero.com/bctl/v1/Server/src/DaemonServerWebsocket/Handlers/HandleREST"
)

// Constructor to create a new Daemon Server Websocket Client
func NewDaemonServerWebsocketClient(serviceURL string, daemonConnectionId string, token string) *DaemonServerWebsocketTypes.DaemonServerWebsocket {
	ret := DaemonServerWebsocketTypes.DaemonServerWebsocket{}

	// First load in our Kube variables
	// TODO: Where should we save this, in the class? is this the best way to do this?
	// TODO: Also we should be able to drop this req, and just load `IN CLUSTER CONFIG`
	serviceAccountTokenPath := os.Getenv("KUBERNETES_SERVICE_ACCOUNT_TOKEN_PATH")
	serviceAccountTokenBytes, _ := ioutil.ReadFile(serviceAccountTokenPath)
	// TODO: Check for error
	serviceAccountToken := string(serviceAccountTokenBytes)
	kubeHost := "https://" + os.Getenv("KUBERNETES_SERVICE_HOST")

	// Create our headers and params, headers are empty
	// TODO: We need to drop this session id auth header req and move to a token based system
	headers := make(map[string]string)

	// Add our token to our params
	params := make(map[string]string)
	params["daemon_connection_id"] = daemonConnectionId
	params["token"] = token

	hubEndpoint := "/api/v1/hub/kube-server"

	// Add our response channels
	ret.RequestForServerChan = make(chan DaemonServerWebsocketTypes.RequestToClusterFromBastionMessage)
	ret.RequestForStartExecChan = make(chan DaemonServerWebsocketTypes.StartExecToClusterFromBastionSignalRMessage)
	ret.ExecStdoutChan = make(chan DaemonServerWebsocketTypes.SendStdoutToDaemonSignalRMessage)
	ret.ExecStdinChannel = make(chan DaemonServerWebsocketTypes.StdinToClusterFromBastionSignalRMessage)
	ret.ExecResizeChannel = make(chan DaemonServerWebsocketTypes.ResizeTerminalToClusterFromBastionSignalRMessage)

	// Create our response channels
	ret.WebsocketClient = CommonWebsocketClient.NewCommonWebsocketClient(serviceURL, hubEndpoint, params, headers)

	// Set up our handler to deal with incoming messages
	// TODO: There has to be a better  way to do this
	go func() {
		for {
			message := <-ret.WebsocketClient.WebsocketMessageChan
			if bytes.Contains(message, []byte("\"target\":\"RequestToClusterFromBastion\"")) {
				log.Printf("Handling incoming RequestToClusterFromBastion message")
				requestToClusterFromBastionSignalRMessage := new(DaemonServerWebsocketTypes.RequestToClusterFromBastionSignalRMessage)

				err := json.Unmarshal(message, requestToClusterFromBastionSignalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling RequestToClusterFromBastion: %s", err)
					return
				}
				// Broadcase this response to our DataToClientChan
				ret.AlertOnRequestForServerChan(requestToClusterFromBastionSignalRMessage.Arguments[0])
			} else if bytes.Contains(message, []byte("\"target\":\"StartExecToClusterFromBastion\"")) {
				log.Printf("Handling incoming StartExecToClusterFromBastion message")
				startExecToClusterFromBastionSignalRMessage := new(DaemonServerWebsocketTypes.StartExecToClusterFromBastionSignalRMessage)

				err := json.Unmarshal(message, startExecToClusterFromBastionSignalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling StartExecToClusterFromBastion: %s", err)
					return
				}
				// Broadcase this response to our RequestForStartExecChan
				ret.AlertOnRequestForStartExecChan(*startExecToClusterFromBastionSignalRMessage)
			} else if bytes.Contains(message, []byte("\"target\":\"SendStdoutToDaemon\"")) {
				log.Printf("Handling incoming SendStdoutToDaemon message")
				sendStdoutToDaemonSignalRMessage := new(DaemonServerWebsocketTypes.SendStdoutToDaemonSignalRMessage)

				err := json.Unmarshal(message, sendStdoutToDaemonSignalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling SendStdoutToDaemon: %s", err)
					return
				}
				// Broadcase this response to our RequestForStartExecChan
				ret.AlertOnExecStdoutChan(*sendStdoutToDaemonSignalRMessage)
			} else if bytes.Contains(message, []byte("\"target\":\"StdinToClusterFromBastion\"")) {
				log.Printf("Handling incoming StdinToClusterFromBastion message")
				stdinToClusterFromBastionSignalRMessage := new(DaemonServerWebsocketTypes.StdinToClusterFromBastionSignalRMessage)

				err := json.Unmarshal(message, stdinToClusterFromBastionSignalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling StdinToClusterFromBastion: %s", err)
					return
				}
				// Broadcase this response to our RequestForStartExecChan
				ret.AlertOnExecStdinChan(*stdinToClusterFromBastionSignalRMessage)
			} else if bytes.Contains(message, []byte("\"target\":\"ResizeTerminalToClusterFromBastion\"")) {
				log.Printf("Handling incoming ResizeTerminalToClusterFromBastion message")
				resizeTerminalToClusterFromBastionSignalRMessage := new(DaemonServerWebsocketTypes.ResizeTerminalToClusterFromBastionSignalRMessage)

				err := json.Unmarshal(message, resizeTerminalToClusterFromBastionSignalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling ResizeTerminalToClusterFromBastion: %s", err)
					return
				}
				// Broadcase this response to our RequestForStartExecChan
				ret.AlertOnExecResizeChan(*resizeTerminalToClusterFromBastionSignalRMessage)
			} else {
				log.Printf("Unhandled incoming message: %s", string(message))
			}
		}
	}()

	// Set up handlers for our channels
	// Handle incoming REST request messages
	go func() {
		for {
			requestForServer := DaemonServerWebsocketTypes.RequestToClusterFromBastionMessage{}
			requestForServer = <-ret.RequestForServerChan
			go HandleREST.HandleREST(requestForServer, serviceAccountToken, kubeHost, &ret)
		}
	}()

	// Handle incoming Exec request messages
	go func() {
		for {
			startExecToClusterSingalRMessage := DaemonServerWebsocketTypes.StartExecToClusterFromBastionSignalRMessage{}
			startExecToClusterSingalRMessage = <-ret.RequestForStartExecChan
			go HandleExec.HandleExec(startExecToClusterSingalRMessage, serviceAccountToken, kubeHost, &ret)
		}
	}()

	return &ret
}
