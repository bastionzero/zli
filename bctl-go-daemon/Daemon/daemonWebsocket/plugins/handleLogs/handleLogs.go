package handleLogs

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"bastionzero.com/bctl/v1/Daemon/daemonWebsocket"
	"bastionzero.com/bctl/v1/Daemon/daemonWebsocket/daemonWebsocketTypes"
)

// Handler for kubectl logs calls that can be proxied
func HandleLogs(w http.ResponseWriter, r *http.Request, commandBeingRun string, logId string, wsClient *daemonWebsocket.DaemonWebsocket) {
	// First extract all the info out of the request
	headers := make(map[string]string)
	for name, values := range r.Header {
		// Loop over all values for the name.
		for _, value := range values {
			headers[name] = value
		}
	}

	// Extract the endpoint along with query params
	endpointWithQuery := r.URL.String()

	// Extract the method
	method := r.Method

	// Now extract the body
	bodyInBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		return
	}

	// requestIdentifier
	requestIdentifier := wsClient.GenerateUniqueIdentifier()

	// Now put our request together and make the request
	dataFromClientMessage := daemonWebsocketTypes.RequestDaemonToBastion{}
	dataFromClientMessage.LogId = logId
	dataFromClientMessage.KubeCommand = commandBeingRun
	dataFromClientMessage.Endpoint = endpointWithQuery
	dataFromClientMessage.Headers = headers
	dataFromClientMessage.Method = method
	dataFromClientMessage.Body = bodyInBytes
	dataFromClientMessage.RequestIdentifier = requestIdentifier
	wsClient.SendRequestLogDaemonToBastion(dataFromClientMessage)

	// Wait for the responses
	receivedRequestIdentifier := -1
	for {
		responseLogBastionToDaemon := daemonWebsocketTypes.ResponseBastionToDaemon{}
		// TODO : There is a race condition here on this chan between this kubectl logs and other invocations of the same command
		// Possible solution; have a dict of identifiers -> channels to direct every msg to the right chan
		responseLogBastionToDaemon = <-wsClient.ResponseLogToDaemonChan

		receivedRequestIdentifier = responseLogBastionToDaemon.RequestIdentifier
		// Ensure that the identifer is correct
		if receivedRequestIdentifier != requestIdentifier {
			// Rebroadbast the message
			wsClient.AlertOnResponseLogToDaemonChan(responseLogBastionToDaemon)
		} else {
			// Now let the kubectl client know the response
			// First do headers
			for name, value := range responseLogBastionToDaemon.Headers {
				if name != "Content-Length" {
					w.Header().Set(name, value)
				}
			}
			
			// Then stream the response to kubectl
			src := bytes.NewReader(responseLogBastionToDaemon.Content)
			_, err = io.Copy(w, src)
			if err != nil {
				log.Printf("Error streaming the log to kubectl: %v", err)
				return
			}
			// This is required, don't touch - not sure why
			flush, ok := w.(http.Flusher)
			if ok {
				flush.Flush()
			}
			// TODO : This needs to be tested
			if responseLogBastionToDaemon.StatusCode != 200 {
				log.Println("ERROR HANDLING LOG SENDNIG 500")
				w.WriteHeader(http.StatusInternalServerError)
			}
		}

	}
}
