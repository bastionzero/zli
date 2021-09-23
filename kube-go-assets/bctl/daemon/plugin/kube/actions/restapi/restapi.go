package restapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	kuberest "bastionzero.com/bctl/v1/bctl/agent/plugin/kube/actions/restapi"
	kubeutils "bastionzero.com/bctl/v1/bctl/daemon/plugin/kube/utils"
	lggr "bastionzero.com/bctl/v1/bzerolib/logger"
	plgn "bastionzero.com/bctl/v1/bzerolib/plugin"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

const (
	action = "kube/restapi"
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
	// First extract the headers out of the request
	headers := kubeutils.GetHeaders(request.Header)

	// Now extract the body
	bodyInBytes, err := kubeutils.GetBodyBytes(request.Body)
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

		for name, values := range apiResponse.Headers {
			for _, value := range values {
				if name != "Content-Length" {
					writer.Header().Set(name, value)
				}
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
