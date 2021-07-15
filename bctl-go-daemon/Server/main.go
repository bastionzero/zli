package server

import (
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"bastionzero.com/bctl-daemon/v1/websocketClient"
	"bastionzero.com/bctl-daemon/v1/websocketClient/websocketClientTypes"
	// "k8s.io/client-go/kubernetes"
	// "k8s.io/client-go/kubernetes/schs`eme"
	// "k8s.io/client-go/rest"
	// "k8s.io/client-go/tools/clientcmd"
	// "k8s.io/client-go/tools/remotecommand"
	// "k8s.io/client-go/util/homedir"
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

	// Try to exec
	// fmt.Println("HERE")
	// urlToExecWith := kubeHost + "/api/v1/namespaces/default/pods/bzero-dev-84bb7dc697-cqkx7/exec?command=%2Fbin%2Fbash&container=bastion&stdin=true&stdout=true&tty=true"
	// podName := "bzero-dev-84bb7dc697-cqkx7"
	// fmt.Println(urlToExecWith)

	// Set up our config
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
	// err = exec.Stream(remotecommand.StreamOptions{
	// 	Stdin:  os.Stdin,
	// 	Stdout: os.Stdout,
	// 	Stderr: os.Stderr,
	// })
	// if err != nil {
	// 	return err
	// }

	// Sleep forever
	// Ref: https://stackoverflow.com/questions/36419054/go-projects-main-goroutine-sleep-forever
	select {}

	return nil

}
