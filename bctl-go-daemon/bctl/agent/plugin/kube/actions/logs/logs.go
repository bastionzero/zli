package logs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	lggr "bastionzero.com/bctl/v1/bzerolib/logger"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type LogAction struct {
	RequestId           string
	serviceAccountToken string
	kubeHost            string
	impersonateGroup    string
	role                string
	streamOutputChannel chan smsg.StreamMessage
	closed              bool
	doneChannel         chan bool
	logger              *lggr.Logger
	ctx                 context.Context
}

type LogSubAction string

const (
	LogData  LogSubAction = "kube/log/stdout"
	LogStart LogSubAction = "kube/log/start"
	LogStop  LogSubAction = "kube/log/stop"
)

func NewLogAction(ctx context.Context, logger *lggr.Logger, serviceAccountToken string, kubeHost string, impersonateGroup string, role string, ch chan smsg.StreamMessage) (*LogAction, error) {
	return &LogAction{
		serviceAccountToken: serviceAccountToken,
		kubeHost:            kubeHost,
		impersonateGroup:    impersonateGroup,
		role:                role,
		streamOutputChannel: ch,
		doneChannel:         make(chan bool),
		closed:              false,
		logger:              logger,
		ctx:                 ctx,
	}, nil
}

func (l *LogAction) Closed() bool {
	return l.closed
}

func (l *LogAction) InputMessageHandler(action string, actionPayload []byte) (string, []byte, error) {
	// TODO: Check request ID matches from startlog
	switch LogSubAction(action) {

	// Start exec message required before anything else
	case LogStart:
		var logActionRequest KubeLogsActionPayload
		if err := json.Unmarshal(actionPayload, &logActionRequest); err != nil {
			rerr := fmt.Errorf("malformed Kube Logs Action payload %v", actionPayload)
			l.logger.Error(rerr)
			return action, []byte{}, rerr
		}

		return l.StartLog(logActionRequest, action)
	case LogStop:
		l.logger.Info("Stopping Log Action")
		l.doneChannel <- true
		return string(LogStop), []byte{}, nil
	default:
		rerr := fmt.Errorf("unhandled log action: %v", action)
		l.logger.Error(rerr)
		return "", []byte{}, rerr
	}
}

func (l *LogAction) StartLog(logActionRequest KubeLogsActionPayload, action string) (string, []byte, error) {

	endpointWithQuery, err := url.Parse(logActionRequest.Endpoint)
	if err != nil {
		l.logger.Error(err)
		return action, []byte{}, err
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
	containerName := queryParams.Get("container")

	// TODO : Here should be added support for as many as possible native kubectl flags through
	// the request's query params
	podLogOptions := v1.PodLogOptions{
		Follow: followFlag,
		// TailLines: &count,
	}

	if containerName != "" {
		podLogOptions.Container = containerName
	}

	// Create our api object
	config, err := rest.InClusterConfig()
	if err != nil {
		rerr := fmt.Errorf("error grabbing cluster config: %s", err)
		l.logger.Error(rerr)
		return action, []byte{}, rerr
	}
	// Add our impersonation information
	config.Impersonate = rest.ImpersonationConfig{
		UserName: l.role,
		Groups:   []string{l.impersonateGroup},
	}
	config.BearerToken = l.serviceAccountToken

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		rerr := fmt.Errorf("kubernetes config error: %s", err)
		l.logger.Error(rerr)
		return action, []byte{}, rerr
	}

	podLogRequest := clientSet.CoreV1().
		Pods(namespace).
		GetLogs(podName, &podLogOptions)

	stream, err := podLogRequest.Stream(l.ctx)
	if err != nil {
		l.logger.Error(err)
		return action, []byte{}, err
	}

	// Subscribe to normal log request
	go func() {
		defer stream.Close()
		for {
			select {
			case <-l.ctx.Done():
				return
			default:
				buf := make([]byte, 2000)
				numBytes, err := stream.Read(buf)
				if numBytes == 0 {
					continue
				}

				if err != nil {
					if err == io.EOF {
						// TODO : EOF should be passed all the way from here to the client
						continue // placeholder
					}
					l.logger.Error(err)
					l.closed = true
					return
				}
				// Stream the response back
				content := base64.StdEncoding.EncodeToString(buf[:numBytes])
				message := smsg.StreamMessage{
					Type:           string(LogData),
					RequestId:      logActionRequest.RequestId,
					LogId:          logActionRequest.LogId,
					SequenceNumber: -1,
					Content:        content,
				}
				l.streamOutputChannel <- message
			}
		}
	}()

	// Subscribe to our done channel
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-l.doneChannel:
				stream.Close()
				cancel()
				return
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
