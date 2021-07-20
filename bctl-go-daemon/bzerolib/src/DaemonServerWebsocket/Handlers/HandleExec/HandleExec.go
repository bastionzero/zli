package HandleExec

import (
	"fmt"
	"log"
	"net/url"

	"bastionzero.com/bctl/v1/Server/src/DaemonServerWebsocket/DaemonServerWebsocketTypes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// TODO: Maybe this should return an error
func HandleExec(requestForStartExecToClusterSingalRMessage DaemonServerWebsocketTypes.RequestForStartExecToClusterSingalRMessage, serviceAccountToken string, kubeHost string, wsClient *DaemonServerWebsocketTypes.DaemonServerWebsocket) {
	requestForStartExecToClusterMessage := DaemonServerWebsocketTypes.RequestForStartExecToClusterMessage{}
	requestForStartExecToClusterMessage = requestForStartExecToClusterSingalRMessage.Arguments[0]

	// Now open up our local exec session
	// podName := "bzero-nabeel-d639d5e2-856b6f49f-vqz8h"

	// Create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	config.Impersonate = rest.ImpersonationConfig{
		UserName: "cwc-dev-developer",
		Groups:   []string{"system:authenticated"},
	}
	config.BearerToken = serviceAccountToken

	// // Build our client
	// restKubeClient, err := kubernetes.NewForConfig(config)
	// if err != nil {
	// 	panic(err.Error())
	// }

	// Build our post request
	// req := restKubeClient.CoreV1().RESTClient().Post().Resource("pods").Name(podName).
	// 	Namespace("default").SubResource("exec")
	// option := &v1.PodExecOptions{
	// 	Container: "bastion",
	// 	Command:   requestForStartExecToClusterMessage.Command,
	// 	Stdin:     true,
	// 	Stdout:    true,
	// 	Stderr:    true,
	// 	TTY:       true,
	// }
	// // if stdin == nil { // TODO?
	// //     option.Stdin = false
	// // }
	// req.VersionedParams(
	// 	option,
	// 	scheme.ParameterCodec,
	// )
	// TODO: Request params need to be send along with this initial request
	// {StartExecToCluster [{[/bin/bash] /api/v1/namespaces/default/pods/bzero-dev-84b449b778-vmrwg/exec 1318}] 1}
	// https://172.20.0.1:443/api/v1/namespaces/default/pods/bzero-dev-84b449b778-vmrwg/exec?command=%2Fbin%2Fbash&container=bastion&stderr=true&stdin=true&stdout=true&tty=true

	execUrl := kubeHost + requestForStartExecToClusterMessage.Endpoint
	execUrlParsed, _ := url.Parse(execUrl)
	fmt.Println(execUrl)

	// Turn it into a SPDY executor
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", execUrlParsed)
	if err != nil {
		log.Println("Error creating Spdy executor")
		return
	}

	// Finally build our streams
	stdoutWriter := NewStdoutWriter(wsClient, requestForStartExecToClusterMessage.RequestIdentifier)
	stdinReader := NewStdinReader(wsClient, requestForStartExecToClusterMessage.RequestIdentifier)
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdinReader,
		Stdout: stdoutWriter,
		Stderr: stdoutWriter,
	})
	if err != nil {
		log.Println("Error creating Spdy stream")
		log.Println(err)
		return
	}
}
