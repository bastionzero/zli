package daemonServerWebsocket

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"bastionzero.com/bctl/v1/Server/Websockets/daemonServerWebsocket/daemonServerWebsocketTypes"
	"bastionzero.com/bctl/v1/Server/Websockets/daemonServerWebsocket/plugins/handleExec"
	"bastionzero.com/bctl/v1/Server/Websockets/daemonServerWebsocket/plugins/handleREST"
	"bastionzero.com/bctl/v1/Server/Websockets/daemonServerWebsocket/plugins/handleLogs"
	"bastionzero.com/bctl/v1/commonWebsocketClient"
)

// Constructor to create a new Daemon Server Websocket Client
func NewDaemonServerWebsocketClient(serviceURL string, daemonConnectionId string, token string) *daemonServerWebsocketTypes.DaemonServerWebsocket {
	ret := daemonServerWebsocketTypes.DaemonServerWebsocket{}

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
	ret.RequestForServerChan = make(chan daemonServerWebsocketTypes.RequestBastionToCluster)
	ret.RequestLogForServerChan = make(chan daemonServerWebsocketTypes.RequestBastionToCluster)
	ret.RequestLogEndForServerChan = make(chan daemonServerWebsocketTypes.RequestBastionToCluster)
	ret.RequestForStartExecChan = make(chan daemonServerWebsocketTypes.StartExecToClusterFromBastionSignalRMessage)
	ret.ExecStdoutChan = make(chan daemonServerWebsocketTypes.SendStdoutToDaemonSignalRMessage)
	ret.ExecStdinChannel = make(chan daemonServerWebsocketTypes.StdinToClusterFromBastionSignalRMessage)
	ret.ExecResizeChannel = make(chan daemonServerWebsocketTypes.ResizeTerminalToClusterFromBastionSignalRMessage)

	// Create our response channels
	ret.WebsocketClient = commonWebsocketClient.NewCommonWebsocketClient(serviceURL, hubEndpoint, params, headers)

	// Make our cancel context, unused for now
	ctx, _ := context.WithCancel(context.Background())

	// Set up our handler to deal with incoming messages
	// TODO: There has to be a better  way to do this
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case message := <-ret.WebsocketClient.WebsocketMessageChan:
				if bytes.Contains(message, []byte("\"target\":\"RequestToClusterFromBastion\"")) {
					log.Printf("Handling incoming RequestToClusterFromBastion message")
					requestToClusterFromBastionSignalRMessage := new(daemonServerWebsocketTypes.RequestBastionToClusterSignalRMessage)

					err := json.Unmarshal(message, requestToClusterFromBastionSignalRMessage)
					if err != nil {
						log.Printf("Error un-marshalling RequestToClusterFromBastion: %s", err)
						return
					}
					// Broadcase this response to our DataToClientChan
					ret.AlertOnRequestForServerChan(requestToClusterFromBastionSignalRMessage.Arguments[0])
				} else if bytes.Contains(message, []byte("\"target\":\"RequestLogToClusterFromBastion\"")) {
					log.Printf("Handling incoming RequestLogToClusterFromBastion message")
					requestLogBastionToClusterSignalRMessage := new(daemonServerWebsocketTypes.RequestBastionToClusterSignalRMessage)

					err := json.Unmarshal(message, requestLogBastionToClusterSignalRMessage)
					if err != nil {
						log.Printf("Error un-marshalling RequestLogBastionToCluster: %s", err)
						return
					}

					// If this is a message to end a log just alert the log end channel
					if (requestLogBastionToClusterSignalRMessage.Arguments[0].End){
						ret.AlertOnRequestLogEndForServerChan(requestLogBastionToClusterSignalRMessage.Arguments[0])
					} else { // Alert the new log request channel
						ret.AlertOnRequestLogForServerChan(requestLogBastionToClusterSignalRMessage.Arguments[0])
					}
				} else if bytes.Contains(message, []byte("\"target\":\"StartExecToClusterFromBastion\"")) {
					log.Printf("Handling incoming StartExecToClusterFromBastion message")
					startExecToClusterFromBastionSignalRMessage := new(daemonServerWebsocketTypes.StartExecToClusterFromBastionSignalRMessage)

					err := json.Unmarshal(message, startExecToClusterFromBastionSignalRMessage)
					if err != nil {
						log.Printf("Error un-marshalling StartExecToClusterFromBastion: %s", err)
						return
					}
					// Broadcase this response to our RequestForStartExecChan
					ret.AlertOnRequestForStartExecChan(*startExecToClusterFromBastionSignalRMessage)
				} else if bytes.Contains(message, []byte("\"target\":\"SendStdoutToDaemon\"")) {
					log.Printf("Handling incoming SendStdoutToDaemon message")
					sendStdoutToDaemonSignalRMessage := new(daemonServerWebsocketTypes.SendStdoutToDaemonSignalRMessage)

					err := json.Unmarshal(message, sendStdoutToDaemonSignalRMessage)
					if err != nil {
						log.Printf("Error un-marshalling SendStdoutToDaemon: %s", err)
						return
					}
					// Broadcase this response to our RequestForStartExecChan
					ret.AlertOnExecStdoutChan(*sendStdoutToDaemonSignalRMessage)
				} else if bytes.Contains(message, []byte("\"target\":\"StdinToClusterFromBastion\"")) {
					log.Printf("Handling incoming StdinToClusterFromBastion message")
					stdinToClusterFromBastionSignalRMessage := new(daemonServerWebsocketTypes.StdinToClusterFromBastionSignalRMessage)

					err := json.Unmarshal(message, stdinToClusterFromBastionSignalRMessage)
					if err != nil {
						log.Printf("Error un-marshalling StdinToClusterFromBastion: %s", err)
						return
					}
					// Broadcase this response to our RequestForStartExecChan
					ret.AlertOnExecStdinChan(*stdinToClusterFromBastionSignalRMessage)
				} else if bytes.Contains(message, []byte("\"target\":\"ResizeTerminalToClusterFromBastion\"")) {
					log.Printf("Handling incoming ResizeTerminalToClusterFromBastion message")
					resizeTerminalToClusterFromBastionSignalRMessage := new(daemonServerWebsocketTypes.ResizeTerminalToClusterFromBastionSignalRMessage)

					err := json.Unmarshal(message, resizeTerminalToClusterFromBastionSignalRMessage)
					if err != nil {
						log.Printf("Error un-marshalling ResizeTerminalToClusterFromBastion: %s", err)
						return
					}
					// Broadcase this response to our RequestForStartExecChan
					ret.AlertOnExecResizeChan(*resizeTerminalToClusterFromBastionSignalRMessage)
				} else if bytes.Contains(message, []byte("\"target\":\"CloseConnectionToClusterFromBastion\"")) {
					log.Printf("Handling incoming CloseConnectionToClusterFromBastion message")
					closeConnectionToClusterFromBastionSignalRMessage := new(daemonServerWebsocketTypes.CloseConnectionToClusterFromBastionSignalRMessage)

					err := json.Unmarshal(message, closeConnectionToClusterFromBastionSignalRMessage)
					if err != nil {
						log.Printf("Error un-marshalling CloseConnectionToClusterFromBastion: %s", err)
						return
					}
					// Close the connection and the go routine
					ret.WebsocketClient.Close = true
					ret.WebsocketClient.Cancel()
					ret.WebsocketClient.Client.Close()

					log.Printf("Closed connection")
				} else {
					log.Printf("Unhandled incoming message: %s", string(message))
				}
			}
		}
	}()

	// Set up handlers for our channels
	// Handle incoming REST request messages
	go func() {
		for {
			requestForServer := daemonServerWebsocketTypes.RequestBastionToCluster{}
			select {
			case <-ctx.Done():
				return
			case requestForServer = <-ret.RequestForServerChan:
				go handleREST.HandleREST(requestForServer, serviceAccountToken, kubeHost, &ret)
			}

		}
	}()

	// Handle incoming Logs request messages
	go func() {
		for {
			requestLogForServer := daemonServerWebsocketTypes.RequestBastionToCluster{}
			select {
			case <-ctx.Done():
				return
			case requestLogForServer = <-ret.RequestLogForServerChan:
				go handleLogs.HandleLogs(requestLogForServer, serviceAccountToken, kubeHost, &ret)
			}

		}
	}()

	// Handle incoming Exec request messages
	go func() {
		for {
			startExecToClusterSingalRMessage := daemonServerWebsocketTypes.StartExecToClusterFromBastionSignalRMessage{}
			select {
			case <-ctx.Done():
				return
			case startExecToClusterSingalRMessage = <-ret.RequestForStartExecChan:
				go handleExec.HandleExec(startExecToClusterSingalRMessage, serviceAccountToken, kubeHost, &ret)
			}
		}
	}()

	return &ret
}
