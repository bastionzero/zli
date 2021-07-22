package handleExec

import (
	"log"
	"net/url"

	"bastionzero.com/bctl/v1/Server/Websockets/daemonServerWebsocket/daemonServerWebsocketTypes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// TODO: Maybe this should return an error
func HandleExec(startExecToClusterFromBastionSignalRMessage daemonServerWebsocketTypes.StartExecToClusterFromBastionSignalRMessage, serviceAccountToken string, kubeHost string, wsClient *daemonServerWebsocketTypes.DaemonServerWebsocket) {
	startExecToClusterMessage := daemonServerWebsocketTypes.StartExecToClusterFromBastionMessage{}
	startExecToClusterMessage = startExecToClusterFromBastionSignalRMessage.Arguments[0]

	// Now open up our local exec session
	// Create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	// Add our impersonation information
	// TODO: Make this not hardcoded, bastion should send info here
	config.Impersonate = rest.ImpersonationConfig{
		UserName: startExecToClusterMessage.Role,
		Groups:   []string{"system:authenticated"},
	}
	config.BearerToken = serviceAccountToken

	execUrl := kubeHost + startExecToClusterMessage.Endpoint
	execUrlParsed, _ := url.Parse(execUrl)

	// Turn it into a SPDY executor
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", execUrlParsed)
	if err != nil {
		log.Println("Error creating Spdy executor")
		return
	}

	// Finally build our streams
	stdoutWriter := NewStdoutWriter(wsClient, startExecToClusterMessage.RequestIdentifier)
	stderrWriter := NewStderrWriter(wsClient, startExecToClusterMessage.RequestIdentifier)
	stdinReader := NewStdinReader(wsClient, startExecToClusterMessage.RequestIdentifier)
	terminalSizeQueue := NewTerminalSizeQueue(wsClient, startExecToClusterMessage.RequestIdentifier)
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:             stdinReader,
		Stdout:            stdoutWriter,
		Stderr:            stderrWriter,
		TerminalSizeQueue: terminalSizeQueue,
		Tty:               true, // TODO: We dont always want tty
	})
	if err != nil {
		log.Println("Error creating Spdy stream")
		log.Println(err)
		return
	}
}
