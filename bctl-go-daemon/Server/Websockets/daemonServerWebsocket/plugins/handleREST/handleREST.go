package handleREST

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"

	"bastionzero.com/bctl/v1/Server/Websockets/daemonServerWebsocket/daemonServerWebsocketTypes"
)

func HandleREST(requestForServer daemonServerWebsocketTypes.RequestBastionToCluster, serviceAccountToken string, kubeHost string, wsClient *daemonServerWebsocketTypes.DaemonServerWebsocket) {
	log.Printf("Handling incoming RequestForServerFromBastion. For endpoint %s", requestForServer.Endpoint)

	// Perform the api request
	httpClient := &http.Client{}
	finalUrl := kubeHost + requestForServer.Endpoint
	bodyBytesReader := bytes.NewReader(requestForServer.Body)
	req, _ := http.NewRequest(requestForServer.Method, finalUrl, bodyBytesReader)

	// Add any headers
	for name, values := range requestForServer.Headers {
		// Loop over all values for the name.
		req.Header.Set(name, values)
	}

	// Add our impersonation and token headers
	req.Header.Set("Authorization", "Bearer "+serviceAccountToken)
	req.Header.Set("Impersonate-User", requestForServer.Role)
	req.Header.Set("Impersonate-Group", "system:authenticated")

	// Make the request and wait for the body to close
	log.Printf("Making request for %s", finalUrl)

	// TODO: Figure out a way around this
	// CA certs can be found here /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	res, err := httpClient.Do(req)
	// TODO: Check for error here
	if err != nil {
		log.Printf("Bad response: %s", err)
		return
	}
	defer res.Body.Close()

	// Now we need to send that data back to the client
	responseToBastionFromCluster := daemonServerWebsocketTypes.ResponseClusterToBastion{}
	responseToBastionFromCluster.StatusCode = res.StatusCode
	responseToBastionFromCluster.RequestIdentifier = requestForServer.RequestIdentifier

	// Build the header response
	header := make(map[string]string)
	for key, value := range res.Header {
		// TODO: This does not seem correct, we should add all headers even if they are dups
		header[key] = value[0]
	}
	responseToBastionFromCluster.Headers = header

	// Parse out the body
	bodyBytes, _ := ioutil.ReadAll(res.Body)
	responseToBastionFromCluster.Content = bodyBytes

	// Finally send our response
	wsClient.SendResponseClusterToBastion(responseToBastionFromCluster) // TODO: This returns err
	// check(err)

	log.Println("Responded to message")
}
