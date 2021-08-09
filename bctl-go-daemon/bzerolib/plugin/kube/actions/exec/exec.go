package exec

import (
	"fmt"
	"net/url"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
	stdin "bastionzero.com/bctl/v1/bzerolib/stream/stdreader"
	stdout "bastionzero.com/bctl/v1/bzerolib/stream/stdwriter"
)

type ExecSubAction string

const (
	StartExec  ExecSubAction = "exec/start"
	ExecInput  ExecSubAction = "exec/input"
	ExecResize ExecSubAction = "exec/resize"
	StopExec   ExecSubAction = "exec/stop"
)

type ExecAction struct {
	ServiceAccountToken string
	KubeHost            string
	ImpersonateGroup    string
	streamOutputChannel chan smsg.StreamMessage
	execInstances       map[string]map[string]interface{}
}

func (r *ExecAction) InputMessageHandler(action string, actionPayload interface{}) (interface{}, error) {
	switch ExecSubAction(action) {
	case StartExec:
		return smsg.StreamMessage{}, r.StartExec(actionPayload)
	case ExecInput:
		break
	case StopExec:
		break
	default:
		return smsg.StreamMessage{}, fmt.Errorf("Recieved unhandled exec action")
	}
	return smsg.StreamMessage{}, nil // We don't need to return a payload with exec
}

func (r *ExecAction) StartExec(payload interface{}) error {
	startExecRequest, ok := payload.(KubeExecStartActionPayload)
	if !ok {
		return fmt.Errorf("Recieved malformed payload")
	}

	// Now open up our local exec session
	// Create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("Error creating in-custer config: %v", err.Error())
	}

	// Add our impersonation information
	// TODO: Make this not hardcoded, bastion should send info here
	config.Impersonate = rest.ImpersonationConfig{
		UserName: startExecRequest.Role,
		Groups:   []string{r.ImpersonateGroup},
	}
	config.BearerToken = r.ServiceAccountToken

	kubeExecApiUrl := r.KubeHost + startExecRequest.Endpoint
	kubeExecApiUrlParsed, _ := url.Parse(kubeExecApiUrl)

	// Turn it into a SPDY executor
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", kubeExecApiUrlParsed)
	if err != nil {
		return fmt.Errorf("Error creating Spdy executor")
	}

	// I've added sequence numbers to stderr and stdout but they don't match up
	// I think this is okay for now, but might be a nice feature in the future? Probably can do it
	// with their own channel + some mutex locks?
	stderrWriter := stdout.NewStdWriter(smsg.StdErr, r.streamOutputChannel, startExecRequest.RequestId)
	stdoutWriter := stdout.NewStdWriter(smsg.StdOut, r.streamOutputChannel, startExecRequest.RequestId)
	stdinReader := stdin.NewStdReader(smsg.StdIn, startExecRequest.RequestId)
	terminalSizeQueue := NewTerminalSizeQueue(startExecRequest.RequestId)

	if err := exec.Stream(remotecommand.StreamOptions{
		Stdin:             stdinReader,
		Stdout:            stdoutWriter,
		Stderr:            stderrWriter,
		TerminalSizeQueue: terminalSizeQueue,
		Tty:               true, // TODO: We dont always want tty
	}); err != nil {
		return fmt.Errorf("Error creating Spdy stream")
	} else {
		return nil
	}
}
