package logs

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	kubewatch "bastionzero.com/bctl/v1/bctl/agent/plugin/kube/actions/watch"
	lggr "bastionzero.com/bctl/v1/bzerolib/logger"
	plgn "bastionzero.com/bctl/v1/bzerolib/plugin"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

const (
	startLogs = "kube/watch/start"
	stopLogs  = "kube/watch/stop"
)

type WatchAction struct {
	requestId             string
	logId                 string
	ksResponseChannel     chan plgn.ActionWrapper
	RequestChannel        chan plgn.ActionWrapper
	streamResponseChannel chan smsg.StreamMessage
	logger                *lggr.Logger
	ctx                   context.Context
}

func NewWatchAction(ctx context.Context,
	logger *lggr.Logger,
	requestId string,
	logId string,
	ch chan plgn.ActionWrapper) (*WatchAction, error) {

	return &WatchAction{
		requestId:             requestId,
		logId:                 logId,
		RequestChannel:        ch,
		ksResponseChannel:     make(chan plgn.ActionWrapper, 100),
		streamResponseChannel: make(chan smsg.StreamMessage, 100),
		logger:                logger,
		ctx:                   ctx,
	}, nil
}

func (r *WatchAction) InputMessageHandler(writer http.ResponseWriter, request *http.Request) error {
	// First extract the headers out of the request
	headers := getHeaders(request.Header)

	// Now extract the body
	bodyInBytes, err := getBodyBytes(request.Body)
	if err != nil {
		r.logger.Error(err)
		return err
	}

	// Build the action payload
	payload := kubewatch.KubeWatchActionPayload{
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

	writer.Header().Set("Content-Type", "application/json")

	// Now subscribe to the response
	// Keep this as a non-go function so we hold onto the http request
	for {
		select {
		case <-r.ctx.Done():
			return nil
		case <-request.Context().Done():
			r.logger.Info(fmt.Sprintf("Watch request %v was requested to get cancelled", r.requestId))

			// Build the action payload
			payload := kubewatch.KubeWatchActionPayload{
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
		case watchData := <-r.streamResponseChannel:
			// Then stream the response to kubectl
			contentBytes, _ := base64.StdEncoding.DecodeString(watchData.Content)
			src := bytes.NewReader(contentBytes)
			_, err = io.Copy(writer, src)
			if err != nil {
				rerr := fmt.Errorf("error streaming the watch to kubectl: %s", err)
				r.logger.Error(rerr)
				break
			}
			// This is required to flush the data to the client
			flush, ok := writer.(http.Flusher)
			if ok {
				flush.Flush()
			}
		}
	}
	return nil
}

func (r *WatchAction) PushKSResponse(wrappedAction plgn.ActionWrapper) {
	r.ksResponseChannel <- wrappedAction
}

func (r *WatchAction) PushStreamResponse(message smsg.StreamMessage) {
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
