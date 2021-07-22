package HandleREST

import (
	"io/ioutil"
	"log"
	"net/http"

	"bastionzero.com/bctl/v1/Daemon/src/DaemonWebsocket"
)

// Handler for Regular Rest calls that can be proxied
func HandleREST(w http.ResponseWriter, r *http.Request, commandBeingRun string, logId string, wsClient *DaemonWebsocket.DaemonWebsocket) {
	// First extract all the info out of the request
	// gzipRequest := false
	headers := make(map[string]string)
	for name, values := range r.Header {
		// Loop over all values for the name.
		for _, value := range values {
			// if name == "Accept-Encoding" && strings.Contains(value, "gzip") {
			// 	gzipRequest = true
			// }
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
	dataFromClientMessage := DaemonWebsocket.RequestToBastionFromDaemonMessage{}
	dataFromClientMessage.LogId = logId
	dataFromClientMessage.KubeCommand = commandBeingRun
	dataFromClientMessage.Endpoint = endpointWithQuery
	dataFromClientMessage.Headers = headers
	dataFromClientMessage.Method = method
	dataFromClientMessage.Body = bodyInBytes
	dataFromClientMessage.RequestIdentifier = requestIdentifier
	wsClient.SendRequestToBastionFromDaemonMessage(dataFromClientMessage)

	// Wait for the response
	receivedRequestIdentifier := -1
	for receivedRequestIdentifier != requestIdentifier {
		responseToDaemonFromBastionMessageResponse := DaemonWebsocket.ResponseToDaemonFromBastionMessage{}
		responseToDaemonFromBastionMessageResponse = <-wsClient.ResponseToDaemonChan

		receivedRequestIdentifier = responseToDaemonFromBastionMessageResponse.RequestIdentifier
		// Ensure that the identifer is correct
		if receivedRequestIdentifier != requestIdentifier {
			// Rebroadbast the message
			wsClient.AlertOnResponseToDaemonChan(responseToDaemonFromBastionMessageResponse)
		} else {
			// Now let the kubectl client know the response
			// First do headers
			for name, value := range responseToDaemonFromBastionMessageResponse.Headers {
				if name != "Content-Length" {
					w.Header().Set(name, value)
				}
			}

			// w.Header().Set("Content-Encoding", "gzip")
			// gz := gzip.NewWriter(w)
			// defer gz.Close()
			// gzr := gzipResponseWriter{Writer: gz, ResponseWriter: w}

			// w.Header().Set("Content-Type", "application/json")
			w.Write(responseToDaemonFromBastionMessageResponse.Content)
			if responseToDaemonFromBastionMessageResponse.StatusCode != 200 {
				log.Println("ERROR HANDLING REST SENDNIG 500")
				w.WriteHeader(http.StatusInternalServerError)
			}
		}

	}
}
