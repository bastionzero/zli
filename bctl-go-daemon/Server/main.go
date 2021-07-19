package main

import (
	"flag"

	"bastionzero.com/bctl/v1/Server/src/ControlWebsocket"
	"bastionzero.com/bctl/v1/Server/src/DaemonServerWebsocket"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// Main ServerInit Thread to execute
// serviceURL string, authHeader string, sessionId string, clientIdentifier string
func main() {
	// TODO: Remove this requirement
	// sessionIdPtr := flag.String("sessionId", "", "Session ID From Zli")
	// authHeaderPtr := flag.String("authHeader", "", "Auth Header From Zli")

	// Our expected flags we need to start
	// TODO: Add a token here that can be used as a way to auth to Bastion,
	// TODO: we should get all these vars also from the env as a backup as we can assume we are inside a container
	serviceURLPtr := flag.String("serviceURL", "", "Service URL to use")

	// Parse and TODO: ensure they all exist
	flag.Parse()

	wsClient := ControlWebsocket.NewControlWebsocketClient(*serviceURLPtr)

	// Subscribe to our handlers
	go func() {
		for {
			message := <-wsClient.ProvisionWebsocketChan

			// We have an incoming websocket request, attempt to make a new Daemon Websocket Client for the request
			token := "1234" // TODO figure this out
			DaemonServerWebsocket.NewDaemonServerWebsocketClient(*serviceURLPtr, message.ConnectionId, token)
		}
	}()

	// // First load in our Kube variables
	// serviceAccountTokenBytes, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	// check(err)
	// serviceAccountToken := string(serviceAccountTokenBytes)
	// kubeHost := "https://" + os.Getenv("KUBERNETES_SERVICE_HOST")

	// // Load in our root CAs
	// // rootCAs, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	// // check(err)

	// // Open a Websocket to Bastion
	// log.Printf("Opening websocket to Bastion: %s", serviceURL)
	// wsClient := websocketClient.NewWebsocketClient(authHeader, sessionId, "", serviceURL, clientIdentifier)

	// // Handle incoming REST request messages
	// go func() {
	// 	for {
	// 		requestForServerSignalRMessage := websocketClientTypes.RequestForServerSignalRMessage{}
	// 		requestForServerSignalRMessage = <-wsClient.RequestForServerChan

	// 		requestForServer := websocketClientTypes.RequestForServerMessage{}
	// 		if len(requestForServerSignalRMessage.Arguments) == 0 {
	// 			break
	// 		}
	// 		requestForServer = requestForServerSignalRMessage.Arguments[0]
	// 		log.Printf("Handling incoming RequestForServer. For endpoint %s", requestForServer.Endpoint)

	// 		// Perform the api request
	// 		httpClient := &http.Client{}
	// 		finalUrl := kubeHost + requestForServer.Endpoint
	// 		req, _ := http.NewRequest(requestForServer.Method, finalUrl, nil)

	// 		// Add any headers
	// 		for name, values := range requestForServer.Headers {
	// 			// Loop over all values for the name.
	// 			req.Header.Set(name, values)
	// 		}

	// 		// Add our impersonation and token headers
	// 		req.Header.Set("Authorization", "Bearer "+serviceAccountToken)
	// 		req.Header.Set("Impersonate-User", "cwc-dev-developer")
	// 		req.Header.Set("Impersonate-Group", "system:authenticated")

	// 		// Make the request and wait for the body to close
	// 		log.Printf("Making request for %s", finalUrl)

	// 		// TODO: Figure out a way around this
	// 		// CA certs can be found here /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
	// 		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// 		res, _ := httpClient.Do(req)
	// 		defer res.Body.Close()

	// 		// Now we need to send that data back to the client
	// 		responseToDaemon := websocketClientTypes.ResponseToDaemonMessage{}
	// 		responseToDaemon.StatusCode = res.StatusCode
	// 		responseToDaemon.RequestIdentifier = requestForServer.RequestIdentifier

	// 		// Build the header response
	// 		header := make(map[string]string)
	// 		for key, value := range res.Header {
	// 			// TODO: This does not seem correct, we should add all headers even if they are dups
	// 			header[key] = value[0]
	// 		}
	// 		responseToDaemon.Headers = header

	// 		// Parse out the body
	// 		bodyBytes, _ := ioutil.ReadAll(res.Body)
	// 		responseToDaemon.Content = string(bodyBytes)

	// 		// Finally send our response
	// 		err = wsClient.SendResponseToDaemonMessage(responseToDaemon)
	// 		check(err)

	// 		log.Println("Responded to message")
	// 	}

	// }()

	// // Handle incoming Exec request messages
	// go func() {
	// 	for {
	// 		requestForStartExecToClusterSingalRMessage := websocketClientTypes.RequestForStartExecToClusterSingalRMessage{}
	// 		requestForStartExecToClusterSingalRMessage = <-wsClient.RequestForStartExecChan
	// 		requestForStartExecToClusterMessage := websocketClientTypes.RequestForStartExecToClusterMessage{}
	// 		requestForStartExecToClusterMessage = requestForStartExecToClusterSingalRMessage.Arguments[0]

	// 		fmt.Println(requestForStartExecToClusterSingalRMessage)
	// 		// Now open up our local exec session
	// 		podName := "bzero-dev-84bb7dc697-cqkx7"

	// 		// Create the in-cluster config
	// 		config, err := rest.InClusterConfig()
	// 		if err != nil {
	// 			panic(err.Error())
	// 		}

	// 		config.Impersonate = rest.ImpersonationConfig{
	// 			UserName: "cwc-dev-developer",
	// 			Groups:   []string{"system:authenticated"},
	// 		}
	// 		config.BearerToken = serviceAccountToken

	// 		// Build our client
	// 		restKubeClient, err := kubernetes.NewForConfig(config)
	// 		if err != nil {
	// 			panic(err.Error())
	// 		}

	// 		// Build our post request
	// 		req := restKubeClient.CoreV1().RESTClient().Post().Resource("pods").Name(podName).
	// 			Namespace("default").SubResource("exec")
	// 		option := &v1.PodExecOptions{
	// 			Container: "bastion",
	// 			Command:   requestForStartExecToClusterMessage.Command,
	// 			Stdin:     true,
	// 			Stdout:    true,
	// 			Stderr:    true,
	// 			TTY:       true,
	// 		}
	// 		// if stdin == nil { // TODO?
	// 		//     option.Stdin = false
	// 		// }
	// 		req.VersionedParams(
	// 			option,
	// 			scheme.ParameterCodec,
	// 		)

	// 		// Turn it into a SPDY executor
	// 		exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	// 		if err != nil {
	// 			log.Println("Error creating Spdy executor")
	// 			return
	// 		}

	// 		// Finally build our streams
	// 		stdoutWriter := NewStdoutWriter(wsClient, requestForStartExecToClusterMessage.RequestIdentifier)
	// 		stdinReader := NewStdinReader(wsClient, requestForStartExecToClusterMessage.RequestIdentifier)
	// 		err = exec.Stream(remotecommand.StreamOptions{
	// 			Stdin:  stdinReader,
	// 			Stdout: stdoutWriter,
	// 			Stderr: stdoutWriter,
	// 		})
	// 		if err != nil {
	// 			log.Println("Error creating Spdy stream")
	// 			log.Println(err)
	// 			return
	// 		}
	// 		time.Sleep(8 * time.Second)
	// 	}
	// }()

	// Sleep forever
	// Ref: https://stackoverflow.com/questions/36419054/go-projects-main-goroutine-sleep-forever
	select {}

}

// Our customer stdout writer so we can pass it into StreamOptions
// type StdoutWriter struct {
// 	ch                chan byte
// 	wsClient          *websocketClient.WebsocketClient
// 	RequestIdentifier int
// }

// // Constructor
// func NewStdoutWriter(wsClient *websocketClient.WebsocketClient, requestIdentifier int) *StdoutWriter {
// 	return &StdoutWriter{
// 		ch:                make(chan byte, 1024),
// 		wsClient:          wsClient,
// 		RequestIdentifier: requestIdentifier,
// 	}
// }

// func (w *StdoutWriter) Chan() <-chan byte {
// 	return w.ch
// }

// // Our custom write function, this will send the data over the websocket
// func (w *StdoutWriter) Write(p []byte) (int, error) {
// 	// Send this data over our websocket
// 	sendStdoutToBastionMessage := &websocketClientTypes.SendStdoutToBastionMessage{}
// 	sendStdoutToBastionMessage.RequestIdentifier = w.RequestIdentifier
// 	sendStdoutToBastionMessage.Stdout = string(p)
// 	w.wsClient.SendSendStdoutToBastionMessage(*sendStdoutToBastionMessage)

// 	// Calculate what needs to be returned
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
// 	wsClient          *websocketClient.WebsocketClient
// 	RequestIdentifier int
// }

// func NewStdinReader(wsClient *websocketClient.WebsocketClient, requestIdentifier int) *StdinReader {
// 	return &StdinReader{
// 		wsClient:          wsClient,
// 		RequestIdentifier: requestIdentifier,
// 	}
// }

// func (r *StdinReader) Read(p []byte) (int, error) {
// 	// time.Sleep(time.Second * 2)
// 	// if r.readIndex >= int64(len(r.data)) {
// 	// 	err = io.EOF
// 	// 	return
// 	// }

// 	// n = copy(p, r.data[r.readIndex:])
// 	// r.readIndex += int64(n)
// 	// return

// 	// I think we will have to manually check for \n or exit, and then return err = io.EOF and n = 0

// 	// First set up our listening for the webscoket
// 	// go func() {
// 	fmt.Println("here before all")
// 	sendStdinToClusterSignalRMessage := websocketClientTypes.SendStdinToClusterSignalRMessage{}
// 	sendStdinToClusterSignalRMessage = <-r.wsClient.ExecStdinChannel
// 	fmt.Println("here before copy")

// 	n := copy(p, []byte(sendStdinToClusterSignalRMessage.Arguments[0].Stdin))
// 	fmt.Println("Here??")

// 	return n, nil

// 	// }()
// }
