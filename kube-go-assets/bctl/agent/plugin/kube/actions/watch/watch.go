package watch

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	kubeutils "bastionzero.com/bctl/v1/bctl/agent/plugin/kube/utils"
	lggr "bastionzero.com/bctl/v1/bzerolib/logger"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

type WatchAction struct {
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

type WatchSubAction string

const (
	WatchData  WatchSubAction = "kube/watch/stdout"
	WatchStart WatchSubAction = "kube/watch/start"
	WatchStop  WatchSubAction = "kube/watch/stop"
)

func NewWatchAction(ctx context.Context, logger *lggr.Logger, serviceAccountToken string, kubeHost string, impersonateGroup string, role string, ch chan smsg.StreamMessage) (*WatchAction, error) {
	return &WatchAction{
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

func (l *WatchAction) Closed() bool {
	return l.closed
}

func (l *WatchAction) InputMessageHandler(action string, actionPayload []byte) (string, []byte, error) {
	switch WatchSubAction(action) {

	// Start exec message required before anything else
	case WatchStart:
		var watchActionRequest KubeWatchActionPayload
		if err := json.Unmarshal(actionPayload, &watchActionRequest); err != nil {
			rerr := fmt.Errorf("malformed Kube Logs Action payload %v", actionPayload)
			l.logger.Error(rerr)
			return action, []byte{}, rerr
		}

		l.requestId = watchActionRequest.RequestId

		return l.StartWatch(watchActionRequest, action)
	case WatchStop:

		var watchActionRequest KubeWatchActionPayload
		if err := json.Unmarshal(actionPayload, &watchActionRequest); err != nil {
			rerr := fmt.Errorf("malformed Kube Watch Action payload %v", actionPayload)
			l.logger.Error(rerr)
			return action, []byte{}, rerr
		}

		if err := l.validateRequestId(watchActionRequest.RequestId); err != nil {
			return "", []byte{}, err
		}

		l.logger.Info("Stopping Watch Action")
		l.doneChannel <- true
		l.closed = true
		return string(WatchStop), []byte{}, nil
	default:
		rerr := fmt.Errorf("unhandled watch action: %v", action)
		l.logger.Error(rerr)
		return "", []byte{}, rerr
	}
}

func (l *WatchAction) validateRequestId(requestId string) error {
	if err := kubeutils.ValidateRequestId(requestId, l.requestId); err != nil {
		l.logger.Error(err)
		return err
	}
	return nil
}

func (w *WatchAction) StartWatch(watchActionRequest KubeWatchActionPayload, action string) (string, []byte, error) {
	// Perform the api request
	httpClient := &http.Client{}
	kubeApiUrl := w.kubeHost + watchActionRequest.Endpoint
	bodyBytesReader := bytes.NewReader([]byte(watchActionRequest.Body))
	req, _ := http.NewRequest(watchActionRequest.Method, kubeApiUrl, bodyBytesReader)

	// Add any headers
	for name, values := range watchActionRequest.Headers {
		// Loop over all values for the name.
		req.Header.Set(name, values)
	}

	// Add our impersonation and token headers
	req.Header.Set("Authorization", "Bearer "+w.serviceAccountToken)
	req.Header.Set("Impersonate-User", w.role)
	req.Header.Set("Impersonate-Group", w.impersonateGroup)

	// Make the request and wait for the body to close
	w.logger.Info(fmt.Sprintf("Making request for %s", kubeApiUrl))

	// TODO: Figure out a way around this
	// CA certs can be found here /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	res, err := httpClient.Do(req)
	if err != nil {
		rerr := fmt.Errorf("bad response to API request: %s", err)
		w.logger.Error(rerr)
		return action, []byte{}, rerr
	}

	// Send our first message with the headers
	headers := make(map[string][]string)
	for name, value := range res.Header {
		headers[name] = value
	}
	kubeWatchHeadersPayload := KubeWatchHeadersPayload{
		Headers: headers,
	}
	kubeWatchHeadersPayloadBytes, _ := json.Marshal(kubeWatchHeadersPayload)
	content := base64.StdEncoding.EncodeToString(kubeWatchHeadersPayloadBytes[:])

	// Stream the response back
	message := smsg.StreamMessage{
		Type:           string(WatchData),
		RequestId:      watchActionRequest.RequestId,
		LogId:          watchActionRequest.LogId,
		SequenceNumber: 0,
		Content:        content,
	}
	w.streamOutputChannel <- message

	buf := make([]byte, WatchBufferSize)

	go func() {
		for {
			select {
			case <-w.ctx.Done():
				return
			default:
				// Read into the buffer
				numBytes, err := res.Body.Read(buf)

				// Check the errors
				if err == io.EOF {
					// This means we are done
					w.logger.Info("Received  EOF error on Watch stream")
					return
				}
				if err != nil {
					w.logger.Info(fmt.Sprintf("Error reading HTTP response: %s", err))
					return
				}

				// Stream the response back
				content := base64.StdEncoding.EncodeToString(buf[:numBytes])
				message := smsg.StreamMessage{
					Type:           string(WatchData),
					RequestId:      watchActionRequest.RequestId,
					LogId:          watchActionRequest.LogId,
					SequenceNumber: -1,
					Content:        content,
				}
				w.streamOutputChannel <- message
			}

		}
	}()

	// Subscribe to our done channel
	go func() {
		for {
			defer res.Body.Close()
			select {
			case <-w.ctx.Done():
				return
			case <-w.doneChannel:
				return
			}
		}
	}()

	return action, []byte{}, nil
}
