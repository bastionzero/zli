package exec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	lggr "bastionzero.com/bctl/v1/bzerolib/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/apimachinery/pkg/util/httpstream/spdy"
)

const (
	// Default timeout for streams
	DefaultStreamCreationTimeout = 30 * time.Second

	// Enable stdin for remote command execution
	ExecStdinParam = "stdin"
	// Enable stdout for remote command execution
	ExecStdoutParam = "stdout"
	// Enable stderr for remote command execution
	ExecStderrParam = "stderr"
	// Enable TTY for remote command execution
	ExecTTYParam = "tty"
	// Command to run for remote command execution
	ExecCommandParam = "command"

	// Name of header that specifies stream type
	StreamType = "streamType"
	// Value for streamType header for stdin stream
	StreamTypeStdin = "stdin"
	// Value for streamType header for stdout stream
	StreamTypeStdout = "stdout"
	// Value for streamType header for stderr stream
	StreamTypeStderr = "stderr"
	// Value for streamType header for data stream
	StreamTypeData = "data"
	// Value for streamType header for error stream
	StreamTypeError = "error"
	// Value for streamType header for terminal resize stream
	StreamTypeResize = "resize"

	// Name of header that specifies the port being forwarded
	PortHeader = "port"
	// Name of header that specifies a request ID used to associate the error
	// and data streams for a single forwarded connection
	PortForwardRequestIDHeader = "requestID"
)

type SPDYService struct {
	conn         io.Closer
	stdinStream  io.ReadCloser
	stdoutStream io.WriteCloser
	stderrStream io.WriteCloser
	writeStatus  func(status *StatusError) error
	resizeStream io.ReadCloser
	// tty          bool
	logger *lggr.Logger
}

type Options struct {
	Stdin           bool
	Stdout          bool
	Stderr          bool
	TTY             bool
	ExpectedStreams int
	Command         []string
}

type streamAndReply struct {
	httpstream.Stream
	replySent <-chan struct{}
}

type StatusError struct {
	ErrStatus metav1.Status
}

func NewSPDYService(logger *lggr.Logger, writer http.ResponseWriter, request *http.Request) (*SPDYService, error) {
	// Extract the options of the exec
	options := extractExecOptions(request)

	logger.Info(fmt.Sprintf("Starting Exec for command: %s\n", options.Command))

	// Initiate a handshake and upgrade the request
	supportedProtocols := []string{"v4.channel.k8s.io", "v3.channel.k8s.io", "v2.channel.k8s.io", "channel.k8s.io"}
	protocol, err := httpstream.Handshake(request, writer, supportedProtocols)
	if err != nil {
		return &SPDYService{}, fmt.Errorf("could not complete http stream handshake: %v", err.Error())
	}
	logger.Info(fmt.Sprintf("Using protocol: %s\n", protocol))

	// Now make our stream channel
	streamCh := make(chan streamAndReply)
	upgrader := spdy.NewResponseUpgrader()
	conn := upgrader.UpgradeResponse(writer, request, func(stream httpstream.Stream, replySent <-chan struct{}) error {
		streamCh <- streamAndReply{Stream: stream, replySent: replySent}
		return nil
	})
	if conn == nil {
		// The upgrader is responsible for notifying the client of any errors that
		// occurred during upgrading. All we can do is return here at this point
		// if we weren't successful in upgrading.
		// TODO: Return a better error
		logger.Error(fmt.Errorf("unable to upgrade request"))
		return &SPDYService{}, fmt.Errorf("unable to upgrade request")
	}

	// Set our idle timeout
	conn.SetIdleTimeout(time.Minute)

	service := &SPDYService{
		conn:   conn,
		logger: logger,
	}

	// Wait for our streams to come in
	expired := time.NewTimer(DefaultStreamCreationTimeout)
	defer expired.Stop()

	// Wait for streams to come in and return SPDY service
	if err := service.waitForStreams(request.Context(), streamCh, options.ExpectedStreams, expired.C); err != nil {
		return &SPDYService{}, fmt.Errorf("error waiting for streams to come in: %v", err.Error())
	} else {
		return service, nil
	}
}

func (s *SPDYService) waitForStreams(connContext context.Context,
	streams <-chan streamAndReply,
	expectedStreams int,
	expired <-chan time.Time) error {

	// Ref: https://github.com/gravitational/teleport/blob/7bff7c41bda0f36898e9063aaacd5539ce062489/lib/kube/proxy/remotecommand.go
	// Make our command object
	receivedStreams := 0
	replyChan := make(chan struct{})
	stopCtx, cancel := context.WithCancel(connContext)
	defer cancel()

WaitForStreams:
	for {
		select {
		// Loop over all incoming strems until we see all expected steams
		case stream := <-streams:
			// Extract the stream type from the header
			streamType := stream.Headers().Get(StreamType)
			s.logger.Info(fmt.Sprintf("Saw stream type: " + streamType))

			// Save the stream
			switch streamType {
			case StreamTypeError:
				s.writeStatus = v4WriteStatusFunc(stream)
				// Send to a buffer to wait, we will wait on replyChan until we see the expected number of streams
				go waitStreamReply(stopCtx, stream.replySent, replyChan)
			case StreamTypeStdin:
				s.stdinStream = stream
				go waitStreamReply(stopCtx, stream.replySent, replyChan)
			case StreamTypeStdout:
				s.stdoutStream = stream
				go waitStreamReply(stopCtx, stream.replySent, replyChan)
			case StreamTypeStderr:
				s.stderrStream = stream
				go waitStreamReply(stopCtx, stream.replySent, replyChan)
			case StreamTypeResize:
				s.resizeStream = stream
				go waitStreamReply(stopCtx, stream.replySent, replyChan)
			default:
				fmt.Printf("Ignoring unexpected stream type: %q", streamType)
			}
		case <-replyChan:
			receivedStreams++
			if receivedStreams == expectedStreams {
				break WaitForStreams
			}
		case <-expired:
			return errors.New("timed out waiting for client to create streams")
		case <-connContext.Done():
			return errors.New("onnectoin has dropped, exiting")
		}
	}
	return nil
}

// v4WriteStatusFunc returns a WriteStatusFunc that marshals a given api Status
// as json in the error channel.
func v4WriteStatusFunc(stream io.Writer) func(status *StatusError) error {
	return func(status *StatusError) error {
		bs, err := json.Marshal(status.ErrStatus)
		if err != nil {
			return err
		}
		_, err = stream.Write(bs)
		return err
	}
}

// waitStreamReply waits until either replySent or stop is closed. If replySent is closed, it sends
// an empty struct to the notify channel.
func waitStreamReply(ctx context.Context, replySent <-chan struct{}, notify chan<- struct{}) {
	select {
	case <-replySent:
		notify <- struct{}{}
	case <-ctx.Done():
	}
}

func extractExecOptions(r *http.Request) Options {
	tty := r.FormValue(ExecTTYParam) == "true"
	stdin := r.FormValue(ExecStdinParam) == "true"
	stdout := r.FormValue(ExecStdoutParam) == "true"
	stderr := r.FormValue(ExecStderrParam) == "true"

	// count the streams client asked for, starting with 1
	expectedStreams := 1
	if stdin {
		expectedStreams++
	}
	if stdout {
		expectedStreams++
	}
	if stderr {
		expectedStreams++
	}
	if tty { // TODO: && handler.supportsTerminalResizing()
		expectedStreams++
	}

	return Options{
		Stdin:           stdin,
		Stdout:          stdout,
		Stderr:          stderr,
		TTY:             tty,
		ExpectedStreams: expectedStreams,
		Command:         r.URL.Query()["command"],
	}
}
