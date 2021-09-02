package logs

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	kubelogs "bastionzero.com/bctl/v1/bctl/agent/plugin/kube/actions/logs"
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
	writer                http.ResponseWriter
	streamResponseChannel chan smsg.StreamMessage
	logger                *lggr.Logger
}

func NewLogAction(logger *lggr.Logger, requestId string, logId string, ch chan plgn.ActionWrapper) (*LogsAction, error) {
	return &LogsAction{
		requestId:             requestId,
		logId:                 logId,
		RequestChannel:        ch,
		ksResponseChannel:     make(chan plgn.ActionWrapper, 100),
		streamResponseChannel: make(chan smsg.StreamMessage, 100),
		logger:                logger,
	}, nil
}

func (r *LogsAction) InputMessageHandler(writer http.ResponseWriter, request *http.Request) error {
	// Set this so that we know how to write the response when we get it later
	r.writer = writer

	// First extract the headers out of the request
	headers := make(map[string]string)
	for name, values := range request.Header {
		for _, value := range values {
			headers[name] = value
		}
	}

	// Now extract the body
	bodyInBytes, err := ioutil.ReadAll(request.Body)
	if err != nil {
		rerr := fmt.Errorf("error building body: %s", err)
		r.logger.Error(rerr)
		return rerr
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
	// Keep this as a non-go function so we hold onto the http request
	for {
		select {
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
			// for name, value := range responseLogBastionToDaemon.Headers {
			// 	if name != "Content-Length" {
			// 		w.Header().Set(name, value)
			// 	}
			// }

			// Then stream the response to kubectl
			contentBytes, _ := base64.StdEncoding.DecodeString(logData.Content)
			src := bytes.NewReader(contentBytes)
			_, err = io.Copy(writer, src)
			if err != nil {
				rerr := fmt.Errorf("error streaming the log to kubectl: %s", err)
				r.logger.Error(rerr)
				break
			}
			// This is required, don't touch - not sure why
			flush, ok := writer.(http.Flusher)
			if ok {
				flush.Flush()
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
