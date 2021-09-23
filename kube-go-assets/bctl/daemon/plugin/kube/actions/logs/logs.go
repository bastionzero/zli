package logs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	kubelogs "bastionzero.com/bctl/v1/bctl/agent/plugin/kube/actions/logs"
	kubeutils "bastionzero.com/bctl/v1/bctl/daemon/plugin/kube/utils"
	lggr "bastionzero.com/bctl/v1/bzerolib/logger"
	plgn "bastionzero.com/bctl/v1/bzerolib/plugin"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

const (
	startLogs = "kube/log/start"
	stopLogs  = "kube/log/stop"
)

type LogsAction struct {
	requestId             string
	logId                 string
	ksResponseChannel     chan plgn.ActionWrapper
	RequestChannel        chan plgn.ActionWrapper
	streamResponseChannel chan smsg.StreamMessage
	logger                *lggr.Logger
	ctx                   context.Context
}

func NewLogAction(ctx context.Context,
	logger *lggr.Logger,
	requestId string,
	logId string,
	ch chan plgn.ActionWrapper) (*LogsAction, error) {

	return &LogsAction{
		requestId:             requestId,
		logId:                 logId,
		RequestChannel:        ch,
		ksResponseChannel:     make(chan plgn.ActionWrapper, 100),
		streamResponseChannel: make(chan smsg.StreamMessage, 100),
		logger:                logger,
		ctx:                   ctx,
	}, nil
}

func (r *LogsAction) InputMessageHandler(writer http.ResponseWriter, request *http.Request) error {
	// First extract the headers out of the request
	headers := getHeaders(request.Header)

	// Now extract the body
	bodyInBytes, err := getBodyBytes(request.Body)
	if err != nil {
		r.logger.Error(err)
		return err
	}

	// Build the action payload
	payload := kubelogs.KubeLogsActionPayload{
		Endpoint:  request.URL.String(),
		Headers:   headers,
		Method:    request.Method,
		Body:      string(bodyInBytes), // fix this
		RequestId: r.requestId,
		LogId:     r.logId,
		End:       false,
	}

	payloadBytes, _ := json.Marshal(payload)
	r.RequestChannel <- plgn.ActionWrapper{
		Action:        startLogs,
		ActionPayload: payloadBytes,
	}

	// Now subscribe to the response
	// Keep this as a non-go routine so we hold onto the http request
	for {
		select {
		case <-r.ctx.Done():
			return nil
		case <-request.Context().Done():
			r.logger.Info(fmt.Sprintf("Logs request %v was requested to get cancelled", r.requestId))

			// Build the action payload
			payload := kubelogs.KubeLogsActionPayload{
				Endpoint:  request.URL.String(),
				Headers:   headers,
				Method:    request.Method,
				Body:      string(bodyInBytes), // fix this
				RequestId: r.requestId,
				LogId:     r.logId,
				End:       true,
			}

			payloadBytes, _ := json.Marshal(payload)
			r.RequestChannel <- plgn.ActionWrapper{
				Action:        stopLogs,
				ActionPayload: payloadBytes,
			}

			return nil
		case logData := <-r.streamResponseChannel:
			// Then stream the response to kubectl
			contentBytes, _ := base64.StdEncoding.DecodeString(logData.Content)
			err := kubeutils.WriteToHttpRequest(contentBytes, writer)
			if err != nil {
				r.logger.Error(err)
				return nil
			}
		}
	}
}

func (r *LogsAction) PushKSResponse(wrappedAction plgn.ActionWrapper) {
	r.ksResponseChannel <- wrappedAction
}

func (r *LogsAction) PushStreamResponse(message smsg.StreamMessage) {
	r.streamResponseChannel <- message
}

// Helper function to extract headers from a http request
func getHeaders(headers http.Header) map[string]string {
	toReturn := make(map[string]string)
	for name, values := range headers {
		for _, value := range values {
			toReturn[name] = value
		}
	}
	return toReturn
}

// Helper function to extract the body of a http request
func getBodyBytes(body io.ReadCloser) ([]byte, error) {
	bodyInBytes, err := ioutil.ReadAll(body)
	if err != nil {
		rerr := fmt.Errorf("error building body: %s", err)
		return nil, rerr
	}
	return bodyInBytes, nil
}
