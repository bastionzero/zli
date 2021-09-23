package stream

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	kubeutils "bastionzero.com/bctl/v1/bctl/agent/plugin/kube/utils"
	lggr "bastionzero.com/bctl/v1/bzerolib/logger"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

type StreamAction struct {
	requestId           string
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

type StreamSubAction string

const (
	StreamData  StreamSubAction = "kube/stream/stdout"
	StreamStart StreamSubAction = "kube/stream/start"
	StreamStop  StreamSubAction = "kube/stream/stop"
)

func NewStreamAction(ctx context.Context, logger *lggr.Logger, serviceAccountToken string, kubeHost string, impersonateGroup string, role string, ch chan smsg.StreamMessage) (*StreamAction, error) {
	return &StreamAction{
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

func (s *StreamAction) Closed() bool {
	return s.closed
}

func (s *StreamAction) InputMessageHandler(action string, actionPayload []byte) (string, []byte, error) {
	switch StreamSubAction(action) {

	// Start exec message required before anything else
	case StreamStart:
		var streamActionRequest KubeStreamActionPayload
		if err := json.Unmarshal(actionPayload, &streamActionRequest); err != nil {
			rerr := fmt.Errorf("malformed Kube Stream Action payload %v", actionPayload)
			s.logger.Error(rerr)
			return action, []byte{}, rerr
		}

		s.requestId = streamActionRequest.RequestId

		return s.StartStream(streamActionRequest, action)
	case StreamStop:
		var streamActionRequest KubeStreamActionPayload
		if err := json.Unmarshal(actionPayload, &streamActionRequest); err != nil {
			rerr := fmt.Errorf("malformed Kube Stream Action payload %v", actionPayload)
			s.logger.Error(rerr)
			return action, []byte{}, rerr
		}
		if err := s.validateRequestId(streamActionRequest.RequestId); err != nil {
			return "", []byte{}, err
		}

		s.logger.Info("Stopping Stream Action")

		s.doneChannel <- true
		s.closed = true
		return string(StreamStop), []byte{}, nil
	default:
		rerr := fmt.Errorf("unhandled stream action: %v", action)
		s.logger.Error(rerr)
		return "", []byte{}, rerr
	}
}

func (s *StreamAction) validateRequestId(requestId string) error {
	if err := kubeutils.ValidateRequestId(requestId, s.requestId); err != nil {
		s.logger.Error(err)
		return err
	}
	return nil
}

func (s *StreamAction) StartStream(streamActionRequest KubeStreamActionPayload, action string) (string, []byte, error) {
	// Build our request
	s.logger.Info(fmt.Sprintf("Making request for %s", streamActionRequest.Endpoint))
	req := s.buildHttpRequest(streamActionRequest.Endpoint, streamActionRequest.Body, streamActionRequest.Method, streamActionRequest.Headers)

	// Make the request and wait for the body to close
	httpClient := &http.Client{}
	res, err := httpClient.Do(req)
	if err != nil {
		rerr := fmt.Errorf("bad response to API request: %s", err)
		s.logger.Error(rerr)
		return action, []byte{}, rerr
	}

	// Send our first message with the headers
	headers := make(map[string][]string)
	for name, value := range res.Header {
		headers[name] = value
	}
	kubeWatchHeadersPayload := KubeStreamHeadersPayload{
		Headers: headers,
	}
	kubeWatchHeadersPayloadBytes, _ := json.Marshal(kubeWatchHeadersPayload)
	content := base64.StdEncoding.EncodeToString(kubeWatchHeadersPayloadBytes[:])

	// Stream the response back
	message := smsg.StreamMessage{
		Type:           string(StreamData),
		RequestId:      streamActionRequest.RequestId,
		LogId:          streamActionRequest.LogId,
		SequenceNumber: 0,
		Content:        content,
	}
	s.streamOutputChannel <- message

	// Create our bufio object
	buf := make([]byte, 1024)
	br := bufio.NewReader(res.Body)

	sequenceNumber := 1

	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
				// Read into the buffer
				numBytes, err := io.ReadFull(br, buf)

				// Check the errors
				if err == io.EOF {
					// This means we are done
					s.logger.Info("Received  EOF error on Watch stream")
					return
				}
				if err != nil {
					s.logger.Info(fmt.Sprintf("Error reading HTTP response: %s", err))
					return
				}

				// Stream the response back
				content := base64.StdEncoding.EncodeToString(buf[:numBytes])
				message := smsg.StreamMessage{
					Type:           string(StreamData),
					RequestId:      streamActionRequest.RequestId,
					LogId:          streamActionRequest.LogId,
					SequenceNumber: sequenceNumber,
					Content:        content,
				}
				s.streamOutputChannel <- message
				sequenceNumber += 1
			}

		}
	}()

	// Subscribe to our done channel
	go func() {
		for {
			defer res.Body.Close()
			select {
			case <-s.ctx.Done():
				return
			case <-s.doneChannel:
				return
			}
		}
	}()

	return action, []byte{}, nil
}

func (s *StreamAction) buildHttpRequest(endpoint, body, method string, headers map[string][]string) *http.Request {
	return kubeutils.BuildHttpRequest(s.kubeHost, endpoint, body, method, headers, s.serviceAccountToken, s.role, s.impersonateGroup)
}
