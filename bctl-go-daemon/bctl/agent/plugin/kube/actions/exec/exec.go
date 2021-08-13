package exec

import (
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

	// To send input/resize to our exec sessions
	execStdinChannel  chan []byte
	execResizeChannel chan KubeExecResizeActionPayload

	logId string
}

func NewExecAction(serviceAccountToken string, kubeHost string, impersonateGroup string, role string, ch chan smsg.StreamMessage, logId string) (*ExecAction, error) {
	return &ExecAction{
		ServiceAccountToken: serviceAccountToken,
		KubeHost:            kubeHost,
		ImpersonateGroup:    impersonateGroup,
		Role:                role,
		streamOutputChannel: ch,
		execStdinChannel:    make(chan []byte, 10),
		execResizeChannel:   make(chan KubeExecResizeActionPayload, 10),
		logId:               logId,
	}, nil
}

func (r *ExecAction) SendStdinToChannel(stdin []byte) (string, []byte, error) {
	r.execStdinChannel <- stdin

	return string(ExecInput), []byte{}, nil
}

func (r *ExecAction) SendResizeToChannel(execResizeAction KubeExecResizeActionPayload) (string, []byte, error) {
	r.execResizeChannel <- execResizeAction

	return string(ExecResize), []byte{}, nil
}

func (r *ExecAction) StartExec(startExecRequest KubeExecStartActionPayload) (string, []byte, error) {
	// Now open up our local exec session
	// Create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return "", []byte{}, fmt.Errorf("Error creating in-custer config: %v", err.Error())
	}

	// Add our impersonation information
	config.Impersonate = rest.ImpersonationConfig{
		UserName: r.Role,
		Groups:   []string{r.ImpersonateGroup},
	}
	config.BearerToken = r.ServiceAccountToken

	kubeExecApiUrl := r.KubeHost + startExecRequest.Endpoint
	kubeExecApiUrlParsed, _ := url.Parse(kubeExecApiUrl)
	log.Println(kubeExecApiUrlParsed)

	// Turn it into a SPDY executor
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", kubeExecApiUrlParsed)
	if err != nil {
		return string(StartExec), []byte{}, fmt.Errorf("Error creating Spdy executor: %v", err.Error())
	}

	// I've added sequence numbers to stderr and stdout but they don't match up
	// I think this is okay for now, but might be a nice feature in the future? Probably can do it
	// with their own channel + some mutex locks?
	stderrWriter := stdout.NewStdWriter(smsg.StdErr, r.streamOutputChannel, startExecRequest.RequestId, r.logId)
	stdoutWriter := stdout.NewStdWriter(smsg.StdOut, r.streamOutputChannel, startExecRequest.RequestId, r.logId)

	// Give our stdinReader a channel to listen for
	stdinReader := stdin.NewStdReader(smsg.StdIn, startExecRequest.RequestId, r.execStdinChannel)

	// Make our terminal size queue
	terminalSizeQueue := NewTerminalSizeQueue(startExecRequest.RequestId, r.execResizeChannel)

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
	}()

	return string(StartExec), []byte{}, nil
}
