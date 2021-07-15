package main

import (
	"flag"
	"log"
	"os"

	daemon "bastionzero.com/bctl-daemon/v1/Daemon"
	server "bastionzero.com/bctl-daemon/v1/Server"
)

func main() {
	// Define our command line args
	sessionIdPtr := flag.String("sessionId", "", "Session ID From Zli")
	assumeRolePtr := flag.String("assumeRole", "", "Assume role if we are running as a Daemon")
	clientIdentifierPtr := flag.String("clientIdentifier", "", "Client Identifier to use if we are running as the server")
	authHeaderPtr := flag.String("authHeader", "", "Auth Header From Zli")
	assumeClusterPtr := flag.String("assumeCluster", "", "Cluster we are trying to connect to") // TODO: This is currently unused
	daemonPortPtr := flag.String("daemonPort", "", "Daemon port we should run on")
	serviceURLPtr := flag.String("serviceURL", "", "Service URL to use")
	if *assumeClusterPtr == "" || *assumeRolePtr == "" || *daemonPortPtr == "" {
	}

	// Try to parse some from the env
	if *sessionIdPtr == "" {
		*sessionIdPtr = os.Getenv("SESSION_ID")
	}

	if *authHeaderPtr == "" {
		*authHeaderPtr = os.Getenv("AUTH_HEADER")
	}
	if *clientIdentifierPtr == "" {
		*clientIdentifierPtr = os.Getenv("CLIENT_IDENTIFIER")
	}
	if *serviceURLPtr == "" {
		*serviceURLPtr = os.Getenv("SERVICE_URL")
	}

	// Parse and ensure they all exist
	flag.Parse()
	if *sessionIdPtr == "" || *serviceURLPtr == "" || *authHeaderPtr == "" {
		// TODO: This should be better parsing
		log.Printf("Error, some required vars not passed \n")
		os.Exit(1)
	}

	if *assumeRolePtr != "" {
		// This means we are the Daemon
		// Need to check for required vars, daemonPort, assumeCluster
		daemon.DaemonInit(*serviceURLPtr, *authHeaderPtr, *sessionIdPtr, *assumeRolePtr, *daemonPortPtr)
	} else if *clientIdentifierPtr != "" {
		server.ServerInit(*serviceURLPtr, *authHeaderPtr, *sessionIdPtr, *clientIdentifierPtr)
	}

	// var kubeconfig *string
	// if home := homedir.HomeDir(); home != "" {
	// 	kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	// } else {
	// 	kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	// }
	// flag.Parse()

	// // use the current context in kubeconfig
	// config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	// if err != nil {
	// 	panic(err.Error())
	// }

	// fmt.Println(config)

	// // Try to exec
	// fmt.Println("HERE")
	// // urlToExecWith := kubeHost + "/api/v1/namespaces/default/pods/bzero-dev-84bb7dc697-cqkx7/exec?command=%2Fbin%2Fbash&container=bastion&stdin=true&stdout=true&tty=true"
	// podName := "bzero-dev-84bb7dc697-cqkx7"
	// // // fmt.Println(urlToExecWith)

	// // // Set up our config
	// // config := &rest.Config{
	// // 	Host: kubeHost,
	// // 	// Insecure: true,
	// // 	Impersonate: rest.ImpersonationConfig{
	// // 		UserName: "cwc-dev-developer",
	// // 	},
	// // 	BearerToken: serviceAccountToken,
	// // }

	// // // config.ContentConfig.GroupVersion = &api.Unversioned
	// // // config.ContentConfig.NegotiatedSerializer = api.Codecs

	// // Build our client
	// restKubeClient, err := kubernetes.NewForConfig(config)
	// if err != nil {
	// 	panic(err.Error())
	// }

	// // Define the command we want to run
	// cmd := []string{
	// 	"/bin/bash",
	// }

	// // Build our post request
	// req := restKubeClient.CoreV1().RESTClient().Post().Resource("pods").Name(podName).
	// 	Namespace("default").SubResource("exec")
	// option := &v1.PodExecOptions{
	// 	Container: "bastion",
	// 	Command:   cmd,
	// 	Stdin:     true,
	// 	Stdout:    true,
	// 	Stderr:    true,
	// 	TTY:       true,
	// }
	// // if stdin == nil {
	// //     option.Stdin = false
	// // }
	// req.VersionedParams(
	// 	option,
	// 	scheme.ParameterCodec,
	// )

	// fmt.Println(req.URL())
	// // Turn it into a SPDY executor
	// exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	// if err != nil {
	// 	return
	// }

	// // Finally build our streams
	// stdoutWriter := NewStdoutWriter()
	// stdinReader := NewStdinReader("test")
	// err = exec.Stream(remotecommand.StreamOptions{
	// 	Stdin:  stdinReader,
	// 	Stdout: stdoutWriter,
	// 	Stderr: stdoutWriter,
	// })

	// if err != nil {
	// 	return
	// }
}

// // Our customer stdout writer so we can pass it into StreamOptions
// type StdoutWriter struct {
// 	ch chan byte
// }

// // Constructor
// func NewStdoutWriter() *StdoutWriter {
// 	return &StdoutWriter{make(chan byte, 1024)}
// }

// func (w *StdoutWriter) Chan() <-chan byte {
// 	return w.ch
// }

// // Our custom write function, this will send the data over the websocket
// func (w *StdoutWriter) Write(p []byte) (int, error) {
// 	log.Println(string(p))
// 	fmt.Println(string(p))
// 	n := 0
// 	for _, b := range p {
// 		w.ch <- b
// 		n++
// 	}
// 	return n, nil
// }

// // Close the writer by closing the channel
// func (w *StdoutWriter) Close() error {
// 	close(w.ch)
// 	return nil
// }

// // Our custom stdin reader so we can pass it into Stream Options
// type StdinReader struct {
// 	data      []byte
// 	readIndex int64
// }

// func NewStdinReader(src string) *StdinReader {
// 	return &StdinReader{data: []byte(src)}
// }

// func (r *StdinReader) Read(p []byte) (n int, err error) {
// 	time.Sleep(time.Second * 2)
// 	if r.readIndex >= int64(len(r.data)) {
// 		err = io.EOF
// 		return
// 	}

// 	n = copy(p, r.data[r.readIndex:])
// 	r.readIndex += int64(n)
// 	return
// }
