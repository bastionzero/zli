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
	ret.RequestForServerChan = make(chan DaemonServerWebsocketTypes.RequestForServerMessage)
	ret.RequestForStartExecChan = make(chan DaemonServerWebsocketTypes.RequestForStartExecToClusterSingalRMessage)
	ret.ExecStdoutChan = make(chan DaemonServerWebsocketTypes.SendStdoutToDaemonSignalRMessage)
	ret.ExecStdinChannel = make(chan DaemonServerWebsocketTypes.SendStdinToClusterSignalRMessage)

	// Create our response channels
	ret.WebsocketClient = CommonWebsocketClient.NewCommonWebsocketClient(serviceURL, hubEndpoint, params, headers)

	// Set up our handler to deal with incoming messages
	go func() {
		for {
			message := <-ret.WebsocketClient.WebsocketMessageChan
			if bytes.Contains(message, []byte("\"target\":\"RequestForServer\"")) {
				log.Printf("Handling incoming RequestForServer message")
				requestForServerSignalRMessage := new(DaemonServerWebsocketTypes.RequestForServerSignalRMessage)

				err := json.Unmarshal(message, requestForServerSignalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling RequestForServerSignalRMessage: %s", err)
					return
				}
				// Broadcase this response to our DataToClientChan
				ret.RequestForServerChan <- requestForServerSignalRMessage.Arguments[0]
			} else if bytes.Contains(message, []byte("\"target\":\"StartExecToCluster\"")) {
				log.Printf("Handling incoming StartExecToCluster message")
				requestForStartExecToClusterSingalRMessage := new(DaemonServerWebsocketTypes.RequestForStartExecToClusterSingalRMessage)

				err := json.Unmarshal(message, requestForStartExecToClusterSingalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling StartExecToCluster: %s", err)
					return
				}
				// Broadcase this response to our RequestForStartExecChan
				ret.RequestForStartExecChan <- *requestForStartExecToClusterSingalRMessage
			} else if bytes.Contains(message, []byte("\"target\":\"SendStdoutToDaemon\"")) {
				log.Printf("Handling incoming SendStdoutToDaemon message")
				sendStdoutToDaemonSignalRMessage := new(DaemonServerWebsocketTypes.SendStdoutToDaemonSignalRMessage)

				err := json.Unmarshal(message, sendStdoutToDaemonSignalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling SendStdoutToDaemon: %s", err)
					return
				}
				// Broadcase this response to our RequestForStartExecChan
				ret.ExecStdoutChan <- *sendStdoutToDaemonSignalRMessage
			} else if bytes.Contains(message, []byte("\"target\":\"SendStdinToCluster\"")) {
				log.Printf("Handling incoming SendStdinToCluster message")
				sendStdinToClusterSignalRMessage := new(DaemonServerWebsocketTypes.SendStdinToClusterSignalRMessage)

				err := json.Unmarshal(message, sendStdinToClusterSignalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling SendStdoutToDaemon: %s", err)
					return
				}
				// Broadcase this response to our RequestForStartExecChan
				ret.ExecStdinChannel <- *sendStdinToClusterSignalRMessage
			} else {

				log.Printf("Unhandled incoming message: %s", string(message))
			}

		}
	}()

	// Set up handlers for our channels
	// Handle incoming REST request messages
	go func() {
		for {
			requestForServer := DaemonServerWebsocketTypes.RequestForServerMessage{}
			requestForServer = <-ret.RequestForServerChan
			HandleREST.HandleREST(requestForServer, serviceAccountToken, kubeHost, &ret)

			// // requestForServer := RequestForServerMessage{}
			// // if len(requestForServerSignalRMessage.Arguments) == 0 {
			// // 	break
			// // }
			// // requestForServer = requestForServerSignalRMessage.Arguments[0]
			// log.Printf("Handling incoming RequestForServer. For endpoint %s", requestForServer.Endpoint)

			// // Perform the api request
			// httpClient := &http.Client{}
			// finalUrl := kubeHost + requestForServer.Endpoint
			// req, _ := http.NewRequest(requestForServer.Method, finalUrl, nil)

			// // Add any headers
			// for name, values := range requestForServer.Headers {
			// 	// Loop over all values for the name.
			// 	req.Header.Set(name, values)
			// }

			// // Add our impersonation and token headers
			// req.Header.Set("Authorization", "Bearer "+serviceAccountToken)
			// req.Header.Set("Impersonate-User", "cwc-dev-developer")
			// req.Header.Set("Impersonate-Group", "system:authenticated")

			// // Make the request and wait for the body to close
			// log.Printf("Making request for %s", finalUrl)

			// // TODO: Figure out a way around this
			// // CA certs can be found here /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
			// http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

			// res, err := httpClient.Do(req)
			// // TODO: Check for error here
			// if err != nil {
			// 	log.Printf("Bad response: %s", err)
			// 	return
			// }
			// defer res.Body.Close()

			// // Now we need to send that data back to the client
			// responseToDaemon := DaemonServerWebsocketTypes.ResponseToDaemonMessage{}
			// responseToDaemon.StatusCode = res.StatusCode
			// responseToDaemon.RequestIdentifier = requestForServer.RequestIdentifier

			// // Build the header response
			// header := make(map[string]string)
			// for key, value := range res.Header {
			// 	// TODO: This does not seem correct, we should add all headers even if they are dups
			// 	header[key] = value[0]
			// }
			// responseToDaemon.Headers = header

			// // Parse out the body
			// bodyBytes, _ := ioutil.ReadAll(res.Body)
			// responseToDaemon.Content = string(bodyBytes)

			// // Finally send our response
			// ret.SendResponseToDaemonMessage(responseToDaemon) // This returns err
			// // check(err)

			// log.Println("Responded to message")
		}

	}()

	// Handle incoming Exec request messages
	go func() {
		for {
			requestForStartExecToClusterSingalRMessage := DaemonServerWebsocketTypes.RequestForStartExecToClusterSingalRMessage{}
			requestForStartExecToClusterSingalRMessage = <-ret.RequestForStartExecChan
			HandleExec.HandleExec(requestForStartExecToClusterSingalRMessage, serviceAccountToken, kubeHost, &ret)
		}
	}()

	return &ret
}
