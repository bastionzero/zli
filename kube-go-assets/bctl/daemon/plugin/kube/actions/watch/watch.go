package logs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	kubewatch "bastionzero.com/bctl/v1/bctl/agent/plugin/kube/actions/watch"
	kubeutils "bastionzero.com/bctl/v1/bctl/daemon/plugin/kube/utils"
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

func (w *WatchAction) InputMessageHandler(writer http.ResponseWriter, request *http.Request) error {
	// First extract the headers out of the request
	headers := kubeutils.GetHeaders(request.Header)

	// Now extract the body
	bodyInBytes, err := kubeutils.GetBodyBytes(request.Body)
	if err != nil {
		w.logger.Error(err)
		return err
	}

	// Build the action payload
	payload := kubewatch.KubeWatchActionPayload{
		Endpoint:  request.URL.String(),
		Headers:   headers,
		Method:    request.Method,
		Body:      string(bodyInBytes), // fix this
		RequestId: w.requestId,
		LogId:     w.logId,
		End:       false,
	}

	payloadBytes, _ := json.Marshal(payload)
	w.RequestChannel <- plgn.ActionWrapper{
		Action:        startLogs,
		ActionPayload: payloadBytes,
	}

	// Wait for our initial message to determine what headers to use
	// The first message that comes from the stream is our headers message, wait for it
	// And keep any other messages that might come before
	earlyMessages := make([][]byte, kubewatch.WatchBufferSize)
	earlyMessageNumber := 0
earlyMessageHandler:
	for {
		select {
		case <-w.ctx.Done():
			return nil
		case watchData := <-w.streamResponseChannel:
			contentBytes, _ := base64.StdEncoding.DecodeString(watchData.Content)

			// Attempt to decode contentBytes
			var kubewatchHeadersPayload kubewatch.KubeWatchHeadersPayload
			if err := json.Unmarshal(contentBytes, &kubewatchHeadersPayload); err != nil {
				// If we see an error this must be an early message
				earlyMessages[earlyMessageNumber] = contentBytes
				earlyMessageNumber += 1
			} else {
				// This is our header message, loop and apply
				for name, values := range kubewatchHeadersPayload.Headers {
					for _, value := range values {
						writer.Header().Set(name, value)
					}
				}
				break earlyMessageHandler
			}
		}
	}

	// If there are any early messages, stream them first
	for _, earlyMessage := range earlyMessages {
		err := kubeutils.WriteToHttpRequest(earlyMessage, writer)
		if err != nil {
			return nil
		}
	}

	// Now subscribe to the response
	// Keep this as a non-go routine so we hold onto the http request
	for {
		select {
		case <-w.ctx.Done():
			return nil
		case <-request.Context().Done():
			w.logger.Info(fmt.Sprintf("Watch request %v was requested to get cancelled", w.requestId))

			// Build the action payload
			payload := kubewatch.KubeWatchActionPayload{
				Endpoint:  request.URL.String(),
				Headers:   headers,
				Method:    request.Method,
				Body:      string(bodyInBytes), // fix this
				RequestId: w.requestId,
				LogId:     w.logId,
				End:       true,
			}

			payloadBytes, _ := json.Marshal(payload)
			w.RequestChannel <- plgn.ActionWrapper{
				Action:        stopLogs,
				ActionPayload: payloadBytes,
			}

			return nil
		case watchData := <-w.streamResponseChannel:
			// Then stream the response to kubectl
			contentBytes, _ := base64.StdEncoding.DecodeString(watchData.Content)
			err := kubeutils.WriteToHttpRequest(contentBytes, writer)
			if err != nil {
				w.logger.Error(err)
				return nil
			}
		}
	}
}

func (w *WatchAction) PushKSResponse(wrappedAction plgn.ActionWrapper) {
	w.ksResponseChannel <- wrappedAction
}

func (w *WatchAction) PushStreamResponse(message smsg.StreamMessage) {
	w.streamResponseChannel <- message
}
