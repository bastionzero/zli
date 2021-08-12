package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"strconv"
	"strings"

	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type LogAction struct {
	serviceAccountToken string
	kubeHost            string
	impersonateGroup    string
	role                string
	streamOutputChannel chan smsg.StreamMessage
}

const (
	LogData = "kube/log"
)

func NewLogAction(serviceAccountToken string, kubeHost string, impersonateGroup string, role string, ch chan smsg.StreamMessage) (*LogAction, error) {
	return &LogAction{
		serviceAccountToken: serviceAccountToken,
		kubeHost:            kubeHost,
		impersonateGroup:    impersonateGroup,
		role:                role,
		streamOutputChannel: ch,
	}, nil
}

func (r *LogAction) InputMessageHandler(action string, actionPayload []byte) (string, []byte, error) {
	var logActionRequest KubeLogsActionPayload
	if err := json.Unmarshal(actionPayload, &logActionRequest); err != nil {
		log.Printf("Error: %v", err.Error())
		return action, []byte{}, fmt.Errorf("Malformed Keysplitting Action payload %v", actionPayload)
	}

	endpointWithQuery, err := url.Parse(logActionRequest.Endpoint)
	if err != nil {
		log.Printf("Error on url.Parse: %s", err)
		return action, []byte{}, fmt.Errorf("Error on url.Parse %v", actionPayload)
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

	// / Perform the api request through the kube sdk
	// Prepare the responses
	// responseLogClusterToBastion := daemonServerWebsocketTypes.ResponseClusterToBastion{}
	// responseLogClusterToBastion.StatusCode = 200
	// responseLogClusterToBastion.RequestIdentifier = requestLogForServer.RequestIdentifier

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
		UserName: r.role,
		Groups:   []string{r.impersonateGroup},
	}
	config.BearerToken = r.serviceAccountToken

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	podLogRequest := clientSet.CoreV1().
		Pods(namespace).
		GetLogs(podName, &podLogOptions)

	stream, err := podLogRequest.Stream(context.TODO())
	if err != nil {
		log.Printf("Error on podLogRequest.Stream: %s", err)
		return action, []byte{}, fmt.Errorf("Error on podLogRequest.Stream: %s", err)
	}

	// Subscribe to normal log request
	go func() {
		defer log.Printf("Exited successfully log streaming for request: %v", logActionRequest.RequestId)
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
				// Stream the response back
				message := smsg.StreamMessage{
					Type:           LogData,
					RequestId:      logActionRequest.RequestId,
					SequenceNumber: -1,
					Content:        buf[:numBytes],
				}
				r.streamOutputChannel <- message

				// responseLogClusterToBastion.Content = buf[:numBytes]
				// wsClient.SendResponseLogClusterToBastion(responseLogClusterToBastion) // TODO: This returns err
			}
		}
	}()

	// We also need to listen if we get a cancel log request

	return action, []byte{}, nil
}

func indexOf(word string, data []string) int {
	for k, v := range data {
		if word == v {
			return k
		}
	}
	return -1
}
