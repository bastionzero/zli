package handleLogs

import (
	"context"
	"io"
	"log"
	"net/url"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"bastionzero.com/bctl/v1/Server/Websockets/daemonServerWebsocket/daemonServerWebsocketTypes"
)

func HandleLogs(requestLogForServer daemonServerWebsocketTypes.RequestBastionToCluster, serviceAccountToken string, kubeHost string, wsClient *daemonServerWebsocketTypes.DaemonServerWebsocket) {
	log.Printf("Handling incoming RequestLogBastionToServer. For endpoint %s", requestLogForServer.Endpoint)

	endpointWithQuery, err := url.Parse(requestLogForServer.Endpoint)
	if err != nil {
		log.Printf("Error on url.Parse: %s", err)
		return
	}

	// TODO : Is this too hacky? Is there a better way to grab the namespace and podName?
	// Extract from the request url the namespace and the pod name
	paths := strings.Split(endpointWithQuery.Path, "/")
	namespaceIndex := indexOf("namespaces", paths)
	namespace := paths[namespaceIndex+1]
	podIndex := indexOf("pods", paths)
	podName := paths[podIndex+1]

	// TODO : Extend this for more query params
	// Add any kubect flags that were past as query params
	queryParams := endpointWithQuery.Query()
	followFlag, _ := strconv.ParseBool(queryParams.Get("follow"))
	// tailLinesNum := queryParams.Get("tailLines")

	// Perform the api request through the kube sdk
	// Prepare the responses
	responseLogClusterToBastion := daemonServerWebsocketTypes.ResponseClusterToBastion{}
	responseLogClusterToBastion.StatusCode = 200
	responseLogClusterToBastion.RequestIdentifier = requestLogForServer.RequestIdentifier

	// Make our cancel context
	ctx, cancel := context.WithCancel(context.Background())

	// TODO : Here should be added support for as many as possible native kubectl flags through
	// the request's query params
	podLogOptions := v1.PodLogOptions{
		Container: "bastion",
		Follow:    followFlag,
		// TailLines: &count,
	}

	// Create our api object
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// Add our impersonation information
	config.Impersonate = rest.ImpersonationConfig{
		UserName: requestLogForServer.Role,
		Groups:   []string{"system:authenticated"},
	}
	config.BearerToken = serviceAccountToken

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	podLogRequest := clientSet.CoreV1().
		Pods(namespace).
		GetLogs(podName, &podLogOptions)

	// Make our cancel context
	ctx, cancel := context.WithCancel(context.Background())

	stream, err := podLogRequest.Stream(context.TODO())
	if err != nil {
		log.Printf("Error on podLogRequest.Stream: %s", err)
		return
	}

	// Subscribe to normal log request
	go func() {
		defer log.Printf("Exited successfully log streaming for request: %v", requestLogForServer.RequestIdentifier)
		defer stream.Close()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				buf := make([]byte, 2000)
				numBytes, err := stream.Read(buf)
				if numBytes == 0 {
					continue
				}
				if err == io.EOF {
					// TODO : EOF should be passed all the way from here to the client
					cancel()
					break
				}
				if err != nil {
					log.Printf("Error on stream.Read: %s", err)
					cancel()
					return
				}
				// message := string(buf[:numBytes])
				// log.Print(message)
				responseLogClusterToBastion.Content = buf[:numBytes]
				wsClient.SendResponseLogClusterToBastion(responseLogClusterToBastion) // TODO: This returns err
			}
		}
	}()

	// Subscribe to our cancel event
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			// If we received a message to end logs
			case newRequestLogEndForServer := <-wsClient.RequestLogEndForServerChan:
				// And that message was directed to another request
				if newRequestLogEndForServer.RequestIdentifier != requestLogForServer.RequestIdentifier {
					// Rebroadcast the message for the right thread
					wsClient.AlertOnRequestLogEndForServerChan(newRequestLogEndForServer)
				} else { // Otherwise, if it was directed to this thread terminate logs
					log.Printf("User sent SIGINT for request: %v", requestLogForServer.RequestIdentifier)
					cancel()
					stream.Close()
					return
				}
			}
		}
	}()
}

func indexOf(word string, data []string) int {
	for k, v := range data {
		if word == v {
			return k
		}
	}
	return -1
}
