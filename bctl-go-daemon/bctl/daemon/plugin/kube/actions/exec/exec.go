package exec

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"

	kubeexec "bastionzero.com/bctl/v1/bctl/agent/plugin/kube/actions/exec"
	plgn "bastionzero.com/bctl/v1/bzerolib/plugin"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

type ExecAction struct {
	requestId         string
	logId             string
	commandBeingRun   string
	ksResponseChannel chan plgn.ActionWrapper
	RequestChannel    chan plgn.ActionWrapper
	streamChannel     chan smsg.StreamMessage
}

func NewExecAction(requestId string, logId string, ch chan plgn.ActionWrapper, streamResponseChannel chan smsg.StreamMessage, commandBeingRun string) (*ExecAction, error) {
	return &ExecAction{
		requestId:         requestId,
		logId:             logId,
		commandBeingRun:   commandBeingRun,
		RequestChannel:    ch,
		ksResponseChannel: make(chan plgn.ActionWrapper),
		streamChannel:     make(chan smsg.StreamMessage, 100),
	}, nil
}

func (r *ExecAction) InputMessageHandler(writer http.ResponseWriter, request *http.Request) error {
	spdy, err := NewSPDYService(writer, request)
	if err != nil {
		return err
	}

	// Now since we made our local connection to kubectl, initiate a connection with Bastion
	r.RequestChannel <- wrapStartPayload(r.requestId, r.logId, request.URL.Query()["command"], request.URL.String())

	// Make our cancel context
	ctx, cancel := context.WithCancel(context.Background())

	// Set up a go function for stdout
	go func() {
		streamQueue := make(map[int]smsg.StreamMessage)
		seqNumber := 0
		for {
			select {
			case <-ctx.Done():
				return
			case streamMessage := <-r.streamChannel:
				contentBytes, _ := base64.StdEncoding.DecodeString(streamMessage.Content)

				// Check for agent-initiated end e.g. user typing 'exit'
				if string(contentBytes) == kubeexec.EndTimes {
					spdy.conn.Close()
					cancel()
				}

				// Check sequence number is correct, if not store it for later
				if streamMessage.SequenceNumber == seqNumber {
					spdy.stdoutStream.Write(contentBytes)
					seqNumber++

					// Process any existing messages that were recieved out of order
					msg, ok := streamQueue[seqNumber]
					for ok {
						moreBytes, _ := base64.StdEncoding.DecodeString(msg.Content)
						spdy.stdoutStream.Write(moreBytes)
						delete(streamQueue, seqNumber)
						seqNumber++
						msg, ok = streamQueue[seqNumber]
					}
				} else {
					streamQueue[streamMessage.SequenceNumber] = streamMessage
				}
			}
		}

	}()

	// Set up a go function for stdin
	go func() {
		buf := make([]byte, 16)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := spdy.stdinStream.Read(buf)
				// Handle error
				if err == io.EOF {
					// TODO: This means to close the stream
					cancel()
				}
				// Send message to agent
				r.RequestChannel <- wrapStdinPayload(r.requestId, r.logId, buf[:n])
			}
		}

	}()

	// Set up a go function for resize
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				decoder := json.NewDecoder(spdy.resizeStream)

				size := TerminalSize{}
				if err := decoder.Decode(&size); err != nil {
					if err == io.EOF {
						cancel()
					} else {
						log.Printf("Error decoding resize message: %s", err.Error())
					}
				} else {
					// Emit this as a new resize event
					r.RequestChannel <- wrapResizePayload(r.requestId, r.logId, size.Width, size.Height)
				}
			}
		}
	}()
	return nil
}

func (r *ExecAction) PushKSResponse(wrappedAction plgn.ActionWrapper) {
	r.ksResponseChannel <- wrappedAction
}

func (r *ExecAction) PushStreamResponse(stream smsg.StreamMessage) {
	r.streamChannel <- stream
}

func wrapStartPayload(requestId string, logId string, command []string, endpoint string) plgn.ActionWrapper {
	payload := kubeexec.KubeExecStartActionPayload{
		RequestId: requestId,
		LogId:     logId,
		Command:   command,
		Endpoint:  endpoint,
	}

	payloadBytes, _ := json.Marshal(payload)
	return plgn.ActionWrapper{
		Action:        string(kubeexec.StartExec),
		ActionPayload: payloadBytes,
	}
}

func wrapResizePayload(requestId string, logId string, width uint16, height uint16) plgn.ActionWrapper {
	payload := kubeexec.KubeExecResizeActionPayload{
		RequestId: requestId,
		LogId:     logId,
		Width:     width,
		Height:    height,
	}

	payloadBytes, _ := json.Marshal(payload)
	return plgn.ActionWrapper{
		Action:        string(kubeexec.ExecResize),
		ActionPayload: payloadBytes,
	}
}

func wrapStdinPayload(requestId string, logId string, stdin []byte) plgn.ActionWrapper {
	payload := kubeexec.KubeStdinActionPayload{
		RequestId: requestId,
		LogId:     logId,
		Stdin:     stdin,
	}

	payloadBytes, _ := json.Marshal(payload)
	return plgn.ActionWrapper{
		Action:        string(kubeexec.ExecInput),
		ActionPayload: payloadBytes,
	}
}
