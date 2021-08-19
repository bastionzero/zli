package exec

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

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
	// Build the action payload
	payload := kubeexec.KubeExecStartActionPayload{
		RequestId: r.requestId,
		LogId:     r.logId,
		Command:   request.URL.Query()["command"],
		Endpoint:  request.URL.String(),
	}

	payloadBytes, _ := json.Marshal(payload)
	r.RequestChannel <- plgn.ActionWrapper{
		Action:        string(kubeexec.StartExec),
		ActionPayload: payloadBytes,
	}

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
				if strings.Contains(string(streamMessage.Content), "exit") {
					// First let stdin know to close the stream for the server
					payload := kubeexec.KubeStdinActionPayload{
						RequestId: r.requestId,
						Stdin:     []byte{},
						LogId:     r.logId,
						End:       true,
					}

					payloadBytes, _ := json.Marshal(payload)
					r.RequestChannel <- plgn.ActionWrapper{
						Action:        string(kubeexec.ExecInput),
						ActionPayload: payloadBytes,
					}

					// Close the connection and the context
					spdy.conn.Close()
					cancel()
				} else {
					if streamMessage.SequenceNumber == seqNumber {
						contentBytes, _ := base64.StdEncoding.DecodeString(streamMessage.Content)
						spdy.stdoutStream.Write(contentBytes)
						seqNumber++
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
				// Now we need to send this stdin to Bastion
				payload := kubeexec.KubeStdinActionPayload{
					RequestId: r.requestId,
					Stdin:     buf[:n],
					LogId:     r.logId,
					End:       false,
				}

				payloadBytes, _ := json.Marshal(payload)
				r.RequestChannel <- plgn.ActionWrapper{
					Action:        string(kubeexec.ExecInput),
					ActionPayload: payloadBytes,
				}
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
					log.Printf("Error decoding resize message: %s", err.Error())
					cancel()
				} else {
					// Emit this as a new resize event
					payload := kubeexec.KubeExecResizeActionPayload{
						RequestId: r.requestId,
						LogId:     r.logId,
						Width:     size.Width,
						Height:    size.Height,
					}

					payloadBytes, _ := json.Marshal(payload)
					r.RequestChannel <- plgn.ActionWrapper{
						Action:        string(kubeexec.ExecResize),
						ActionPayload: payloadBytes,
					}
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
