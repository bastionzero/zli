package exec

import (
	"encoding/base64"
	"encoding/json"
	"errors"
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

type ExecAction struct {
	ServiceAccountToken string
	KubeHost            string
	ImpersonateGroup    string
	Role                string
	streamOutputChannel chan smsg.StreamMessage
	// execInstances       map[string]map[string]interface{}
}

func NewExecAction(serviceAccountToken string, kubeHost string, impersonateGroup string, role string, ch chan smsg.StreamMessage) (*ExecAction, error) {
	return &ExecAction{
		ServiceAccountToken: serviceAccountToken,
		KubeHost:            kubeHost,
		ImpersonateGroup:    impersonateGroup,
		Role:                role,
		streamOutputChannel: ch,
	}, nil
}

func (r *ExecAction) InputMessageHandler(action string, actionPayload []byte) (string, []byte, error) {
	switch ExecSubAction(action) {
	case StartExec:
		return r.StartExec(actionPayload)
	case ExecInput:
		break
	case StopExec:
		break
	default:
		return "", []byte{}, errors.New("Recieved unhandled exec action")
	}
	return "", []byte{}, nil // We don't need to return a payload with exec
}

func (r *ExecAction) StartExec(actionPayload []byte) (string, []byte, error) {
	// TODO: The below line removes the extra, surrounding quotation marks that get added at some point in the marshal/unmarshal
	// so it messes up the umarshalling into a valid action payload.  We need to figure out why this is happening
	// so that we can murder its family
	actionPayload = actionPayload[1 : len(actionPayload)-1]

	// Json unmarshalling encodes bytes in base64
	safety, _ := base64.StdEncoding.DecodeString(string(actionPayload))

	var startExecRequest KubeExecStartActionPayload
	if err := json.Unmarshal(safety, &startExecRequest); err != nil {
		log.Printf("Error: %v", err.Error())
		return string(StartExec), []byte{}, fmt.Errorf("Malformed Keysplitting Action payload %v", actionPayload)
	}

	// Now open up our local exec session
	// Create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return string(StartExec), []byte{}, fmt.Errorf("Error creating in-custer config: %v", err.Error())
	}

	// Add our impersonation information
	config.Impersonate = rest.ImpersonationConfig{
		UserName: r.Role,
		Groups:   []string{r.ImpersonateGroup},
	}
	config.BearerToken = r.ServiceAccountToken

	kubeExecApiUrl := r.KubeHost + startExecRequest.Endpoint
	kubeExecApiUrlParsed, _ := url.Parse(kubeExecApiUrl)

	// Turn it into a SPDY executor
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", kubeExecApiUrlParsed)
	if err != nil {
		return string(StartExec), []byte{}, fmt.Errorf("Error creating Spdy executor: %v", err.Error())
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
		return string(StartExec), []byte{}, fmt.Errorf("Error creating Spdy stream: %v", err.Error())
	}

	return string(StartExec), []byte{}, nil
}
