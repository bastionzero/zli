package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	kubeutils "bastionzero.com/bctl/v1/bctl/agent/plugin/kube/utils"
	lggr "bastionzero.com/bctl/v1/bzerolib/logger"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
	stdin "bastionzero.com/bctl/v1/bzerolib/stream/stdreader"
	stdout "bastionzero.com/bctl/v1/bzerolib/stream/stdwriter"
)

type ExecSubAction string

const (
	StartExec  ExecSubAction = "kube/exec/start"
	ExecInput  ExecSubAction = "kube/exec/input"
	ExecResize ExecSubAction = "kube/exec/resize"
	StopExec   ExecSubAction = "kube/exec/stop"
)

const (
	EscChar = "^[" // ESC char
)

type ExecAction struct {
	serviceAccountToken string
	kubeHost            string
	impersonateGroup    string
	role                string
	logId               string
	requestId           string
	closed              bool
	logger              *lggr.Logger
	ctx                 context.Context

	// output channel to send all of our stream messages directly to datachannel
	streamOutputChannel chan smsg.StreamMessage

	// To send input/resize to our exec sessions
	execStdinChannel  chan []byte
	execResizeChannel chan KubeExecResizeActionPayload
}

func NewExecAction(ctx context.Context,
	logger *lggr.Logger,
	serviceAccountToken string,
	kubeHost string,
	impersonateGroup string,
	role string,
	ch chan smsg.StreamMessage) (*ExecAction, error) {

	return &ExecAction{
		serviceAccountToken: serviceAccountToken,
		kubeHost:            kubeHost,
		impersonateGroup:    impersonateGroup,
		role:                role,
		closed:              false,
		streamOutputChannel: ch,
		execStdinChannel:    make(chan []byte, 10),
		execResizeChannel:   make(chan KubeExecResizeActionPayload, 10),
		logger:              logger,
		ctx:                 ctx,
	}, nil
}

func (e *ExecAction) Closed() bool {
	return e.closed
}

func (e *ExecAction) InputMessageHandler(action string, actionPayload []byte) (string, []byte, error) {
	// TODO: Check request ID matches from startexec
	switch ExecSubAction(action) {

	// Start exec message required before anything else
	case StartExec:
		var startExecRequest KubeExecStartActionPayload
		if err := json.Unmarshal(actionPayload, &startExecRequest); err != nil {
			rerr := fmt.Errorf("unable to unmarshal start exec message: %s", err)
			e.logger.Error(rerr)
			return "", []byte{}, rerr
		}

		e.logId = startExecRequest.LogId
		e.requestId = startExecRequest.RequestId
		return e.StartExec(startExecRequest)

	case ExecInput:
		var execInputAction KubeStdinActionPayload
		if err := json.Unmarshal(actionPayload, &execInputAction); err != nil {
			rerr := fmt.Errorf("error unmarshaling stdin: %s", err)
			e.logger.Error(rerr)
			return "", []byte{}, rerr
		}

		if err := e.validateRequestId(execInputAction.RequestId); err != nil {
			return "", []byte{}, err
		}

		e.execStdinChannel <- execInputAction.Stdin
		return string(ExecInput), []byte{}, nil

	case ExecResize:
		var execResizeAction KubeExecResizeActionPayload
		if err := json.Unmarshal(actionPayload, &execResizeAction); err != nil {
			rerr := fmt.Errorf("error unmarshaling resize message: %s", err)
			e.logger.Error(rerr)
			return "", []byte{}, rerr
		}

		if err := e.validateRequestId(execResizeAction.RequestId); err != nil {
			return "", []byte{}, err
		}

		e.execResizeChannel <- execResizeAction
		return string(ExecResize), []byte{}, nil

	default:
		rerr := fmt.Errorf("unhandled exec action: %v", action)
		e.logger.Error(rerr)
		return "", []byte{}, rerr
	}
}

func (e *ExecAction) validateRequestId(requestId string) error {
	if err := kubeutils.ValidateRequestId(requestId, e.requestId); err != nil {
		e.logger.Error(err)
		return err
	}
	return nil
}

func (e *ExecAction) StartExec(startExecRequest KubeExecStartActionPayload) (string, []byte, error) {
	// Now open up our local exec session
	// Create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		rerr := fmt.Errorf("error creating in-custer config: %s", err)
		e.logger.Error(rerr)
		return "", []byte{}, rerr
	}

	// Add our impersonation information
	config.Impersonate = rest.ImpersonationConfig{
		UserName: e.role,
		Groups:   []string{e.impersonateGroup},
	}
	config.BearerToken = e.serviceAccountToken

	kubeExecApiUrl := e.kubeHost + startExecRequest.Endpoint
	kubeExecApiUrlParsed, err := url.Parse(kubeExecApiUrl)
	if err != nil {
		rerr := fmt.Errorf("could not parse kube exec url: %s", err)
		e.logger.Error(rerr)
		return "", []byte{}, rerr
	}

	// Turn it into a SPDY executor
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", kubeExecApiUrlParsed)
	if err != nil {
		return string(StartExec), []byte{}, fmt.Errorf("error creating Spdy executor: %s", err)
	}

	stderrWriter := stdout.NewStdWriter(smsg.StdErr, e.streamOutputChannel, startExecRequest.RequestId, e.logId)
	stdoutWriter := stdout.NewStdWriter(smsg.StdOut, e.streamOutputChannel, startExecRequest.RequestId, e.logId)
	stdinReader := stdin.NewStdReader(smsg.StdIn, startExecRequest.RequestId, e.execStdinChannel)
	terminalSizeQueue := NewTerminalSizeQueue(startExecRequest.RequestId, e.execResizeChannel)

	go func() {
		// This function listens for a closed datachannel.  If the datachannel is closed, it doesn't necessarily mean
		// that the exec was properly closed, and because the below exec.Stream only returns when it's done, there's
		// no way to interrupt it or pass in ctx. Therefore, we need to close the stream in order to pass an io.EOF message
		// to exec which will close the exec.Stream and that will close the go routine.
		// https://github.com/kubernetes/client-go/issues/554
		<-e.ctx.Done()
		stdinReader.Close()
	}()

	go func() {
		if startExecRequest.IsTty {
			err = exec.Stream(remotecommand.StreamOptions{
				Stdin:             stdinReader,
				Stdout:            stdoutWriter,
				Stderr:            stderrWriter,
				TerminalSizeQueue: terminalSizeQueue,
				Tty:               true,
			})
		} else {
			err = exec.Stream(remotecommand.StreamOptions{
				Stdin:  stdinReader,
				Stdout: stdoutWriter,
				Stderr: stderrWriter,
			})
		}

		// First check the error to bubble up to the user
		if err != nil {
			// Log the error
			rerr := fmt.Errorf("error in SPDY stream: %s", err)
			e.logger.Error(rerr)

			// Also write the error to our stdoutWriter so the user can see it
			stdoutWriter.Write([]byte(fmt.Sprint(err)))
		}

		// Now close the stream
		stdoutWriter.Write([]byte(EscChar))

		e.closed = true
	}()

	return string(StartExec), []byte{}, nil
}
