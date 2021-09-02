package exec

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

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
	EndTimes = "^[" // ESC char
)

type ExecAction struct {
	serviceAccountToken string
	kubeHost            string
	impersonateGroup    string
	role                string
	logId               string
	closed              bool

	// output channel to send all of our stream messages directly to datachannel
	streamOutputChannel chan smsg.StreamMessage

	// To send input/resize to our exec sessions
	execStdinChannel  chan []byte
	execResizeChannel chan KubeExecResizeActionPayload
}

func NewExecAction(serviceAccountToken string, kubeHost string, impersonateGroup string, role string, ch chan smsg.StreamMessage) (*ExecAction, error) {
	return &ExecAction{
		serviceAccountToken: serviceAccountToken,
		kubeHost:            kubeHost,
		impersonateGroup:    impersonateGroup,
		role:                role,
		closed:              false,
		streamOutputChannel: ch,
		execStdinChannel:    make(chan []byte, 10),
		execResizeChannel:   make(chan KubeExecResizeActionPayload, 10),
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
			return "", []byte{}, fmt.Errorf("unable to unmarshal start message: %v", err.Error())
		}

		e.logId = startExecRequest.LogId
		return e.StartExec(startExecRequest)

	case ExecInput:
		var execInputAction KubeStdinActionPayload
		if err := json.Unmarshal(actionPayload, &execInputAction); err != nil {
			log.Printf("Error unmarshaling input: %v", err.Error())
			return "", []byte{}, fmt.Errorf("unable to unmarshal stdin message")
		}

		e.execStdinChannel <- execInputAction.Stdin
		return string(ExecInput), []byte{}, nil

	case ExecResize:
		var execResizeAction KubeExecResizeActionPayload
		if err := json.Unmarshal(actionPayload, &execResizeAction); err != nil {
			log.Printf("Error unmarshaling resize: %v", err.Error())
			return "", []byte{}, fmt.Errorf("unable to unmarshal resize message")
		}

		e.execResizeChannel <- execResizeAction
		return string(ExecResize), []byte{}, nil

	default:
		return "", []byte{}, fmt.Errorf("unhandled exec action: %v", action)
	}
}

func (e *ExecAction) StartExec(startExecRequest KubeExecStartActionPayload) (string, []byte, error) {
	// Now open up our local exec session
	// Create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return "", []byte{}, fmt.Errorf("error creating in-custer config: %v", err.Error())
	}

	// Add our impersonation information
	config.Impersonate = rest.ImpersonationConfig{
		UserName: e.role,
		Groups:   []string{e.impersonateGroup},
	}
	config.BearerToken = e.serviceAccountToken

	kubeExecApiUrl := e.kubeHost + startExecRequest.Endpoint
	kubeExecApiUrlParsed, _ := url.Parse(kubeExecApiUrl)
	log.Println(kubeExecApiUrlParsed)

	// Turn it into a SPDY executor
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", kubeExecApiUrlParsed)
	if err != nil {
		return string(StartExec), []byte{}, fmt.Errorf("error creating Spdy executor: %v", err.Error())
	}

	stderrWriter := stdout.NewStdWriter(smsg.StdErr, e.streamOutputChannel, startExecRequest.RequestId, e.logId)
	stdoutWriter := stdout.NewStdWriter(smsg.StdOut, e.streamOutputChannel, startExecRequest.RequestId, e.logId)
	stdinReader := stdin.NewStdReader(smsg.StdIn, startExecRequest.RequestId, e.execStdinChannel)
	terminalSizeQueue := NewTerminalSizeQueue(startExecRequest.RequestId, e.execResizeChannel)

	go func() {
		err := exec.Stream(remotecommand.StreamOptions{
			Stdin:             stdinReader,
			Stdout:            stdoutWriter,
			Stderr:            stderrWriter,
			TerminalSizeQueue: terminalSizeQueue,
			Tty:               true, // TODO: We dont always want tty
		})
		if err != nil {
			// TODO: handle error, send end to daemon
			log.Println("Error with spdy stream")
		}
		stdoutWriter.Write([]byte(EndTimes))
		log.Printf("Spdy stream has ended")
	}()

	return string(StartExec), []byte{}, nil
}
