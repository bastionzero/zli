package restapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	kuberest "bastionzero.com/bctl/v1/bctl/agent/plugin/kube/actions/restapi"
	lggr "bastionzero.com/bctl/v1/bzerolib/logger"
	plgn "bastionzero.com/bctl/v1/bzerolib/plugin"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

const (
	action = "kube/restapi"
)

type RestApiAction struct {
	requestId         string
	logId             string
	ksResponseChannel chan plgn.ActionWrapper
	RequestChannel    chan plgn.ActionWrapper
	writer            http.ResponseWriter
	commandBeingRun   string
	logger            *lggr.Logger
}

func NewRestApiAction(logger *lggr.Logger,
	requestId string,
	logId string,
	ch chan plgn.ActionWrapper,
	commandBeingRun string) (*RestApiAction, error) {

	return &RestApiAction{
		requestId:         requestId,
		logId:             logId,
		RequestChannel:    ch,
		ksResponseChannel: make(chan plgn.ActionWrapper),
		commandBeingRun:   commandBeingRun,
		logger:            logger,
	}, nil
}

func (r *RestApiAction) InputMessageHandler(writer http.ResponseWriter, request *http.Request) error {
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
	case rsp := <-r.ksResponseChannel:
		var apiResponse kuberest.KubeRestApiActionResponsePayload
		if err := json.Unmarshal(rsp.ActionPayload, &apiResponse); err != nil {
			rerr := fmt.Errorf("could not unmarshal Action Response Payload: %s", err)
			r.logger.Error(rerr)
			return rerr
		}

		for name, value := range apiResponse.Headers {
			if name != "Content-Length" {
				r.writer.Header().Set(name, value)
			}
		}

		// output, _ := base64.StdEncoding.DecodeString(string(apiResponse.Content))
		r.writer.Write(apiResponse.Content)
		if apiResponse.StatusCode != 200 {
			r.writer.WriteHeader(http.StatusInternalServerError)

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
}
