package restapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	kubelogs "bastionzero.com/bctl/v1/bctl/agent/plugin/kube/actions/logs"
	kuberest "bastionzero.com/bctl/v1/bctl/agent/plugin/kube/actions/restapi"
	lggr "bastionzero.com/bctl/v1/bzerolib/logger"
	plgn "bastionzero.com/bctl/v1/bzerolib/plugin"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

const (
	action = "kube/restapi"

	startLogs = "kube/log/start"
	stopLogs  = "kube/log/stop"
)

type RestApiAction struct {
	requestId             string
	logId                 string
	ksResponseChannel     chan plgn.ActionWrapper
	RequestChannel        chan plgn.ActionWrapper
	commandBeingRun       string
	streamResponseChannel chan smsg.StreamMessage
	logger                *lggr.Logger
	ctx                   context.Context
}

func NewRestApiAction(ctx context.Context,
	logger *lggr.Logger,
	requestId string,
	logId string,
	ch chan plgn.ActionWrapper,
	streamResponseChannel chan smsg.StreamMessage,
	commandBeingRun string) (*RestApiAction, error) {

	return &RestApiAction{
		requestId:             requestId,
		logId:                 logId,
		RequestChannel:        ch,
		ksResponseChannel:     make(chan plgn.ActionWrapper),
		streamResponseChannel: make(chan smsg.StreamMessage, 100),
		commandBeingRun:       commandBeingRun,
		logger:                logger,
		ctx:                   ctx,
	}, nil
}

func (r *RestApiAction) InputMessageHandler(writer http.ResponseWriter, request *http.Request) error {
	// Determin what type of request this is (regular rest, exec, etc)
	if strings.HasSuffix(request.URL.Path, "/log") {
		return r.handleLogRequest(writer, request)
	} else {
		return r.handleRestRequest(writer, request)
	}
}
func (r *RestApiAction) handleLogRequest(writer http.ResponseWriter, request *http.Request) error {
	// Determin if we are trying to follow the logs
	follow, ok := request.URL.Query()["follow"]

	if !ok || len(follow[0]) < 1 || follow[0] != "true" {
		// If the follow bool is not there, treat this like a regular rest request
		return r.handleRestRequest(writer, request)
	}

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
	// Keep this as a non-go function so we hold onto the http request
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
			src := bytes.NewReader(contentBytes)
			_, err = io.Copy(writer, src)
			if err != nil {
				rerr := fmt.Errorf("error streaming the log to kubectl: %s", err)
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
}

func (r *RestApiAction) handleRestRequest(writer http.ResponseWriter, request *http.Request) error {
	// First extract the headers out of the request
	headers := getHeaders(request.Header)

	// Now extract the body
	bodyInBytes, err := getBodyBytes(request.Body)
	if err != nil {
		r.logger.Error(err)
		return err
	}

	// Build the action payload
	payload := kuberest.KubeRestApiActionPayload{
		Endpoint:        request.URL.String(),
		Headers:         headers,
		Method:          request.Method,
		Body:            string(bodyInBytes), // fix this
		RequestId:       r.requestId,
		LogId:           r.logId,
		CommandBeingRun: r.commandBeingRun,
	}

	payloadBytes, _ := json.Marshal(payload)
	r.RequestChannel <- plgn.ActionWrapper{
		Action:        action,
		ActionPayload: payloadBytes,
	}

	select {
	case <-r.ctx.Done():
		return nil
	case rsp := <-r.ksResponseChannel:
		var apiResponse kuberest.KubeRestApiActionResponsePayload
		if err := json.Unmarshal(rsp.ActionPayload, &apiResponse); err != nil {
			rerr := fmt.Errorf("could not unmarshal Action Response Payload: %s", err)
			r.logger.Error(rerr)
			return rerr
		}

		for name, value := range apiResponse.Headers {
			if name != "Content-Length" {
				writer.Header().Set(name, value)
			}
		}

		// output, _ := base64.StdEncoding.DecodeString(string(apiResponse.Content))
		writer.Write(apiResponse.Content)
		if apiResponse.StatusCode != 200 {
			writer.WriteHeader(http.StatusInternalServerError)

			// log.Printf("ApiResponse Content: %v vs the base64 content: %v", string(apiResponse.Content), string(output))
			rerr := fmt.Errorf("request failed with status code %v: %v", apiResponse.StatusCode, string(apiResponse.Content))
			r.logger.Error(rerr)
			return rerr
		}
	}

	return nil
}

func (r *RestApiAction) PushKSResponse(wrappedAction plgn.ActionWrapper) {
	r.ksResponseChannel <- wrappedAction
}

func (r *RestApiAction) PushStreamResponse(message smsg.StreamMessage) {
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
