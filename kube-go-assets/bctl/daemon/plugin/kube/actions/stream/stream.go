package stream

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	kubestream "bastionzero.com/bctl/v1/bctl/agent/plugin/kube/actions/stream"
	kubeutils "bastionzero.com/bctl/v1/bctl/daemon/plugin/kube/utils"
	lggr "bastionzero.com/bctl/v1/bzerolib/logger"
	plgn "bastionzero.com/bctl/v1/bzerolib/plugin"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

const (
	startStream = "kube/stream/start"
	stopStream  = "kube/stream/stop"
)

type StreamAction struct {
	requestId             string
	logId                 string
	ksResponseChannel     chan plgn.ActionWrapper
	RequestChannel        chan plgn.ActionWrapper
	streamResponseChannel chan smsg.StreamMessage
	logger                *lggr.Logger
	ctx                   context.Context
	commandBeingRun       string
}

func NewStreamAction(ctx context.Context,
	logger *lggr.Logger,
	requestId string,
	logId string,
	ch chan plgn.ActionWrapper,
	commandBeingRun string) (*StreamAction, error) {

	return &StreamAction{
		requestId:             requestId,
		logId:                 logId,
		RequestChannel:        ch,
		ksResponseChannel:     make(chan plgn.ActionWrapper, 100),
		streamResponseChannel: make(chan smsg.StreamMessage, 100),
		logger:                logger,
		ctx:                   ctx,
		commandBeingRun:       commandBeingRun,
	}, nil
}

func (s *StreamAction) InputMessageHandler(writer http.ResponseWriter, request *http.Request) error {
	// First extract the headers out of the request
	headers := kubeutils.GetHeaders(request.Header)

	// Now extract the body
	bodyInBytes, err := kubeutils.GetBodyBytes(request.Body)
	if err != nil {
		s.logger.Error(err)
		return err
	}

	// Build the action payload
	payload := kubestream.KubeStreamActionPayload{
		Endpoint:        request.URL.String(),
		Headers:         headers,
		Method:          request.Method,
		Body:            string(bodyInBytes), // fix this
		RequestId:       s.requestId,
		LogId:           s.logId,
		End:             false,
		CommandBeingRun: s.commandBeingRun,
	}

	payloadBytes, _ := json.Marshal(payload)
	s.RequestChannel <- plgn.ActionWrapper{
		Action:        startStream,
		ActionPayload: payloadBytes,
	}

	// Wait for our initial message to determine what headers to use
	// The first message that comes from the stream is our headers message, wait for it
	// And keep any other messages that might come before
	earlyMessages := make(map[int]smsg.StreamMessage)
earlyMessageHandler:
	for {
		select {
		case <-s.ctx.Done():
			return nil
		case watchData := <-s.streamResponseChannel:
			contentBytes, _ := base64.StdEncoding.DecodeString(watchData.Content)

			// Attempt to decode contentBytes
			var kubelogsHeadersPayload kubestream.KubeStreamHeadersPayload
			if err := json.Unmarshal(contentBytes, &kubelogsHeadersPayload); err != nil {
				// If we see an error this must be an early message
				earlyMessages[watchData.SequenceNumber] = watchData
			} else {
				// This is our header message, loop and apply
				for name, values := range kubelogsHeadersPayload.Headers {
					for _, value := range values {
						writer.Header().Set(name, value)
					}
				}
				break earlyMessageHandler
			}
		}
	}

	// If there are any early messages, stream them first if the sequence number matches
	// Start at 1 since we wait for our headers message
	seqNumber := 1
	for {
		earlyMessageData, ok := earlyMessages[seqNumber]
		for ok {
			// If we have an early message, show it to the user
			contentBytes, _ := base64.StdEncoding.DecodeString(earlyMessageData.Content)
			err := kubeutils.WriteToHttpRequest(contentBytes, writer)
			if err != nil {
				return nil
			}

			// Increment the seqNumber and keep looking for more
			seqNumber += 1
			earlyMessageData, ok = earlyMessages[seqNumber]
		}
		break
	}

	// Now subscribe to the response
	// Keep this as a non-go routine so we hold onto the http request
	for {
		select {
		case <-s.ctx.Done():
			return nil
		case <-request.Context().Done():
			s.logger.Info(fmt.Sprintf("Watch request %v was requested to get cancelled", s.requestId))

			// Build the action payload
			payload := kubestream.KubeStreamActionPayload{
				Endpoint:  request.URL.String(),
				Headers:   headers,
				Method:    request.Method,
				Body:      string(bodyInBytes), // fix this
				RequestId: s.requestId,
				LogId:     s.logId,
				End:       true,
			}

			payloadBytes, _ := json.Marshal(payload)
			s.RequestChannel <- plgn.ActionWrapper{
				Action:        stopStream,
				ActionPayload: payloadBytes,
			}

			return nil
		case watchData := <-s.streamResponseChannel:
			// Then stream the response to kubectl
			if watchData.SequenceNumber == seqNumber {
				// If the incoming data is equal to the current expected seqNumber, show the user
				contentBytes, _ := base64.StdEncoding.DecodeString(watchData.Content)
				err := kubeutils.WriteToHttpRequest(contentBytes, writer)
				if err != nil {
					s.logger.Error(err)
					return nil
				}

				// Increment the seqNumber
				seqNumber += 1

				// See if we have any early messages for this seqNumber
				earlyMessageData, ok := earlyMessages[seqNumber]
				contentBytes, _ = base64.StdEncoding.DecodeString(earlyMessageData.Content)
				for ok {
					// If we do, keep displaying the ones we got early
					err := kubeutils.WriteToHttpRequest(contentBytes, writer)
					if err != nil {
						return nil
					}
					seqNumber += 1
					earlyMessageData, ok = earlyMessages[seqNumber]
				}
			} else {
				earlyMessages[watchData.SequenceNumber] = watchData
			}

		}
	}
}

func (s *StreamAction) PushKSResponse(wrappedAction plgn.ActionWrapper) {
	s.ksResponseChannel <- wrappedAction
}

func (s *StreamAction) PushStreamResponse(message smsg.StreamMessage) {
	s.streamResponseChannel <- message
}
