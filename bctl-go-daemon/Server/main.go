package server

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"bastionzero.com/bctl-daemon/v1/websocketClient"
	"bastionzero.com/bctl-daemon/v1/websocketClient/websocketClientTypes"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// Main Daemon Thread to execute
func ServerInit(serviceURL string, authHeader string, sessionId string, clientIdentifier string) error {
	// First load in our Kube variables
	serviceAccountTokenBytes, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	check(err)
	serviceAccountToken := string(serviceAccountTokenBytes)
	kubeHost := "https://" + os.Getenv("KUBERNETES_SERVICE_HOST")

	// Load in our root CAs
	// rootCAs, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	// check(err)

	// Open a Websocket to Bastion
	log.Printf("Opening websocket to Bastion: %s", serviceURL)
	wsClient := websocketClient.NewWebsocketClient(authHeader, sessionId, "", serviceURL, clientIdentifier)

	// Handle incoming REST request messages
	go func() {
		for {
			requestForServerSignalRMessage := websocketClientTypes.RequestForServerSignalRMessage{}
			requestForServerSignalRMessage = <-wsClient.RequestForServerChan

			requestForServer := websocketClientTypes.RequestForServerMessage{}
			if len(requestForServerSignalRMessage.Arguments) == 0 {
				break
			}
			requestForServer = requestForServerSignalRMessage.Arguments[0]
			log.Printf("Handling incoming RequestForServer. For endpoint %s", requestForServer.Endpoint)

			// Perform the api request
			httpClient := &http.Client{}
			finalUrl := kubeHost + requestForServer.Endpoint
			req, _ := http.NewRequest(requestForServer.Method, finalUrl, nil)

			// Add any headers
			for name, values := range requestForServer.Headers {
				// Loop over all values for the name.
				req.Header.Set(name, values)
			}

			// Add our impersonation and token headers
			req.Header.Set("Authorization", "Bearer "+serviceAccountToken)
			req.Header.Set("Impersonate-User", "cwc-dev-developer")
			req.Header.Set("Impersonate-Group", "system:authenticated")

			// Make the request and wait for the body to close
			log.Printf("Making request for %s", finalUrl)

			// TODO: Figure out a way around this
			// CA certs can be found here /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
			http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

			res, _ := httpClient.Do(req)
			defer res.Body.Close()

			// Now we need to send that data back to the client
			responseToDaemon := websocketClientTypes.ResponseToDaemonMessage{}
			responseToDaemon.StatusCode = res.StatusCode
			responseToDaemon.RequestIdentifier = requestForServer.RequestIdentifier

			// Build the header response
			header := make(map[string]string)
			for key, value := range res.Header {
				// TODO: This does not seem correct, we should add all headers even if they are dups
				header[key] = value[0]
			}
			responseToDaemon.Headers = header

			// Parse out the body
			bodyBytes, _ := ioutil.ReadAll(res.Body)
			responseToDaemon.Content = string(bodyBytes)

			// Finally send our response
			err = wsClient.SendResponseToDaemonMessage(responseToDaemon)
			check(err)

			log.Println("Responded to message")
		}

	}()

	// Handle incoming Exec request messages
	go func() {
		for {
			requestForStartExecToClusterSingalRMessage := websocketClientTypes.RequestForStartExecToClusterSingalRMessage{}
			requestForStartExecToClusterSingalRMessage = <-wsClient.RequestForStartExecChan
			requestForStartExecToClusterMessage := websocketClientTypes.RequestForStartExecToClusterMessage{}
			requestForStartExecToClusterMessage = requestForStartExecToClusterSingalRMessage.Arguments[0]

			fmt.Println(requestForStartExecToClusterSingalRMessage)
			// Now open up our local exec session
			podName := "bzero-dev-84bb7dc697-cqkx7"

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

			// Build our client
			restKubeClient, err := kubernetes.NewForConfig(config)
			if err != nil {
				panic(err.Error())
			}

			// Build our post request
			req := restKubeClient.CoreV1().RESTClient().Post().Resource("pods").Name(podName).
				Namespace("default").SubResource("exec")
			option := &v1.PodExecOptions{
				Container: "bastion",
				Command:   requestForStartExecToClusterMessage.Command,
				Stdin:     true,
				Stdout:    true,
				Stderr:    true,
				TTY:       true,
			}
			// if stdin == nil { // TODO?
			//     option.Stdin = false
			// }
			req.VersionedParams(
				option,
				scheme.ParameterCodec,
			)

			// Turn it into a SPDY executor
			exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
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
			time.Sleep(8 * time.Second)
		}
	}()

	// Handle incoming exec requests

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

	// // Set up our config
	// config := &rest.Config{
	// 	Host: kubeHost,
	// 	// Insecure: true,
	// 	Impersonate: rest.ImpersonationConfig{
	// 		UserName: "cwc-dev-developer",
	// 	},
	// 	BearerToken: serviceAccountToken,
	// }

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
	// 	return err
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
	// 	return err
	// }

	// // Sleep forever
	// // Ref: https://stackoverflow.com/questions/36419054/go-projects-main-goroutine-sleep-forever
	select {}

	return nil

}

// Our customer stdout writer so we can pass it into StreamOptions
type StdoutWriter struct {
	ch                chan byte
	wsClient          *websocketClient.WebsocketClient
	RequestIdentifier int
}

// Constructor
func NewStdoutWriter(wsClient *websocketClient.WebsocketClient, requestIdentifier int) *StdoutWriter {
	return &StdoutWriter{
		ch:                make(chan byte, 1024),
		wsClient:          wsClient,
		RequestIdentifier: requestIdentifier,
	}
}

func (w *StdoutWriter) Chan() <-chan byte {
	return w.ch
}

// Our custom write function, this will send the data over the websocket
func (w *StdoutWriter) Write(p []byte) (int, error) {
	// Send this data over our websocket
	sendStdoutToBastionMessage := &websocketClientTypes.SendStdoutToBastionMessage{}
	sendStdoutToBastionMessage.RequestIdentifier = w.RequestIdentifier
	sendStdoutToBastionMessage.Stdout = string(p)
	w.wsClient.SendSendStdoutToBastionMessage(*sendStdoutToBastionMessage)

	// Calculate what needs to be returned
	n := 0
	for _, b := range p {
		w.ch <- b
		n++
	}
	return n, nil
}

// Close the writer by closing the channel
func (w *StdoutWriter) Close() error {
	close(w.ch)
	return nil
}

// Our custom stdin reader so we can pass it into Stream Options
type StdinReader struct {
	wsClient          *websocketClient.WebsocketClient
	RequestIdentifier int
}

func NewStdinReader(wsClient *websocketClient.WebsocketClient, requestIdentifier int) *StdinReader {
	return &StdinReader{
		wsClient:          wsClient,
		RequestIdentifier: requestIdentifier,
	}
}

func (r *StdinReader) Read(p []byte) (int, error) {
	// time.Sleep(time.Second * 2)
	// if r.readIndex >= int64(len(r.data)) {
	// 	err = io.EOF
	// 	return
	// }

	// n = copy(p, r.data[r.readIndex:])
	// r.readIndex += int64(n)
	// return

	// I think we will have to manually check for \n or exit, and then return err = io.EOF and n = 0

	// First set up our listening for the webscoket
	// go func() {
	fmt.Println("here before all")
	sendStdinToClusterSignalRMessage := websocketClientTypes.SendStdinToClusterSignalRMessage{}
	sendStdinToClusterSignalRMessage = <-r.wsClient.ExecStdinChannel
	fmt.Println("here before copy")

	n := copy(p, []byte(sendStdinToClusterSignalRMessage.Arguments[0].Stdin))
	fmt.Println("Here??")

	return n, nil

	// }()
}
