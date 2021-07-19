package DaemonServerWebsocket

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"bastionzero.com/bctl/v1/CommonWebsocketClient"
	"github.com/gorilla/websocket"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type DaemonServerWebsocket struct {
	WebsocketClient *CommonWebsocketClient.WebsocketClient

	// These are all the    types of channels we have available
	// Basic REST Call related
	RequestForServerChan chan RequestForServerMessage

	// Exec Related
	RequestForStartExecChan chan RequestForStartExecToClusterSingalRMessage
	ExecStdoutChan          chan SendStdoutToDaemonSignalRMessage
	// RequestForServerChan    chan CommonWebsocketClient.RequestForServerSignalRMessage
	// RequestForStartExecChan chan CommonWebsocketClient.RequestForStartExecToClusterSingalRMessage
	ExecStdinChannel chan SendStdinToClusterSignalRMessage

	SocketLock sync.Mutex // Ref: https://github.com/gorilla/websocket/issues/119#issuecomment-198710015
}

// Constructor to create a new Daemon Server Websocket Client
func NewDaemonServerWebsocketClient(serviceURL string, daemonConnectionId string, token string) *DaemonServerWebsocket {
	ret := DaemonServerWebsocket{}

	// First load in our Kube variables
	// TODO: Where should we save this, in the class? is this the best way to do this?
	serviceAccountTokenBytes, _ := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	// TODO: Check for error
	serviceAccountToken := string(serviceAccountTokenBytes)
	kubeHost := "https://" + os.Getenv("KUBERNETES_SERVICE_HOST")

	// Create our headers and params, headers are empty
	// TODO: We need to drop this session id auth header req and move to a token based system
	headers := make(map[string]string)

	// Add our token to our params
	params := make(map[string]string)
	params["daemon_connection_id"] = daemonConnectionId
	params["token"] = token

	hubEndpoint := "/api/v1/hub/kube-server"

	// Add our response channels
	ret.RequestForServerChan = make(chan RequestForServerMessage)
	ret.RequestForStartExecChan = make(chan RequestForStartExecToClusterSingalRMessage)
	ret.ExecStdoutChan = make(chan SendStdoutToDaemonSignalRMessage)
	ret.ExecStdinChannel = make(chan SendStdinToClusterSignalRMessage)

	// Create our response channels
	ret.WebsocketClient = CommonWebsocketClient.NewCommonWebsocketClient(serviceURL, hubEndpoint, params, headers)

	// Set up our handler to deal with incoming messages
	go func() {
		for {
			message := <-ret.WebsocketClient.WebsocketMessageChan
			if bytes.Contains(message, []byte("\"target\":\"RequestForServer\"")) {
				log.Printf("Handling incoming RequestForServer message")
				requestForServerSignalRMessage := new(RequestForServerSignalRMessage)

				err := json.Unmarshal(message, requestForServerSignalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling RequestForServerSignalRMessage: %s", err)
					return
				}
				// Broadcase this response to our DataToClientChan
				ret.RequestForServerChan <- requestForServerSignalRMessage.Arguments[0]
			} else if bytes.Contains(message, []byte("\"target\":\"StartExecToCluster\"")) {
				log.Printf("Handling incoming StartExecToCluster message")
				requestForStartExecToClusterSingalRMessage := new(RequestForStartExecToClusterSingalRMessage)

				err := json.Unmarshal(message, requestForStartExecToClusterSingalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling StartExecToCluster: %s", err)
					return
				}
				// Broadcase this response to our RequestForStartExecChan
				ret.RequestForStartExecChan <- *requestForStartExecToClusterSingalRMessage
			} else if bytes.Contains(message, []byte("\"target\":\"SendStdoutToDaemon\"")) {
				log.Printf("Handling incoming SendStdoutToDaemon message")
				sendStdoutToDaemonSignalRMessage := new(SendStdoutToDaemonSignalRMessage)

				err := json.Unmarshal(message, sendStdoutToDaemonSignalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling SendStdoutToDaemon: %s", err)
					return
				}
				// Broadcase this response to our RequestForStartExecChan
				ret.ExecStdoutChan <- *sendStdoutToDaemonSignalRMessage
			} else if bytes.Contains(message, []byte("\"target\":\"SendStdinToCluster\"")) {
				log.Printf("Handling incoming SendStdinToCluster message")
				sendStdinToClusterSignalRMessage := new(SendStdinToClusterSignalRMessage)

				err := json.Unmarshal(message, sendStdinToClusterSignalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling SendStdoutToDaemon: %s", err)
					return
				}
				// Broadcase this response to our RequestForStartExecChan
				ret.ExecStdinChannel <- *sendStdinToClusterSignalRMessage
			} else {

				log.Printf("Unhandled incoming message: %s", string(message))
			}

		}
	}()

	// Set up handlers for our channels
	// Handle incoming REST request messages
	go func() {
		for {
			requestForServer := RequestForServerMessage{}
			requestForServer = <-ret.RequestForServerChan

			// requestForServer := RequestForServerMessage{}
			// if len(requestForServerSignalRMessage.Arguments) == 0 {
			// 	break
			// }
			// requestForServer = requestForServerSignalRMessage.Arguments[0]
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
			responseToDaemon := ResponseToDaemonMessage{}
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
			ret.SendResponseToDaemonMessage(responseToDaemon) // This returns err
			// check(err)

			log.Println("Responded to message")
		}

	}()

	// Handle incoming Exec request messages
	go func() {
		for {
			requestForStartExecToClusterSingalRMessage := RequestForStartExecToClusterSingalRMessage{}
			requestForStartExecToClusterSingalRMessage = <-ret.RequestForStartExecChan
			requestForStartExecToClusterMessage := RequestForStartExecToClusterMessage{}
			requestForStartExecToClusterMessage = requestForStartExecToClusterSingalRMessage.Arguments[0]

			fmt.Println(requestForStartExecToClusterSingalRMessage)
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
			stdoutWriter := NewStdoutWriter(&ret, requestForStartExecToClusterMessage.RequestIdentifier)
			stdinReader := NewStdinReader(&ret, requestForStartExecToClusterMessage.RequestIdentifier)
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

	return &ret
}

func (client *DaemonServerWebsocket) SendResponseToDaemonMessage(responseToDaemonMessage ResponseToDaemonMessage) error {
	// Lock our mutex and setup the unlock
	client.SocketLock.Lock()
	defer client.SocketLock.Unlock()

	log.Printf("Sending data to Daemon")
	// Create the object, add relevent information
	toSend := new(ResponseToDaemonSignalRMessage)
	toSend.Target = "ResponseToDaemon"
	toSend.Arguments = []ResponseToDaemonMessage{responseToDaemonMessage}

	// Add the type number from the class
	toSend.Type = 1 // Ref: https://github.com/aspnet/SignalR/blob/master/specs/HubProtocol.md#invocation-message-encoding

	// Marshal our message
	toSendMarshalled, err := json.Marshal(toSend)
	if err != nil {
		return err
	}

	// Write our message
	if err = client.WebsocketClient.Client.WriteMessage(websocket.TextMessage, append(toSendMarshalled, 0x1E)); err != nil {
		return err
	}
	// client.SignalRTypeNumber++
	return nil
}

func (client *DaemonServerWebsocket) SendSendStdoutToBastionMessage(sendStdoutToBastionMessage SendStdoutToBastionMessage) error {
	// Lock our mutex and setup the unlock
	client.SocketLock.Lock()
	defer client.SocketLock.Unlock()

	log.Printf("Sending stdout to Cluster")
	// Create the object, add relevent information
	toSend := new(SendStdoutToBastionSignalRMessage)
	toSend.Target = "SendStdoutToBastionFromCluster"
	toSend.Arguments = []SendStdoutToBastionMessage{sendStdoutToBastionMessage}

	// Add the type number from the class
	toSend.Type = 1 // Ref: https://github.com/aspnet/SignalR/blob/master/specs/HubProtocol.md#invocation-message-encoding

	// Marshal our message
	toSendMarshalled, err := json.Marshal(toSend)
	if err != nil {
		return err
	}

	// Write our message
	if err = client.WebsocketClient.Client.WriteMessage(websocket.TextMessage, append(toSendMarshalled, 0x1E)); err != nil {
		log.Printf("Something went wrong :(")
		return err
	}
	// client.SignalRTypeNumber++
	return nil
}
