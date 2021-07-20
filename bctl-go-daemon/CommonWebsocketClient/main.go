package CommonWebsocketClient

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"

	"github.com/gorilla/websocket"
)

// This will be the client that we use to store our websocket connection
type WebsocketClient struct {
	Client            *websocket.Conn
	IsReady           bool
	SignalRTypeNumber int

	SocketLock sync.Mutex // Ref: https://github.com/gorilla/websocket/issues/119#issuecomment-198710015

	// This will be our one response channel whenever we get a websocket message
	WebsocketMessageChan chan []byte

	// These are all the types of channels we have available
	// DataToClientChan        chan DataToClientMessage
	// RequestForServerChan    chan RequestForServerSignalRMessage
	// RequestForStartExecChan chan RequestForStartExecToClusterSingalRMessage
	// ExecStdoutChannel       chan SendStdoutToDaemonSignalRMessage
	// ExecStdinChannel        chan SendStdinToClusterSignalRMessage
}

// All SignalR Messages are teminated with this byte
const messageTerminator byte = 0x1E

// Constructor to create a new common websocket client object that can be shared by the daemon and server
func NewCommonWebsocketClient(serviceUrl string, hubEndpoint string, params map[string]string, headers map[string]string) *WebsocketClient {

	ret := WebsocketClient{}

	// First negotiate in order to get a url to connect to
	httpClient := &http.Client{}
	negotiateUrl := "https://" + serviceUrl + hubEndpoint + "/negotiate"
	req, _ := http.NewRequest("POST", negotiateUrl, nil)

	// Add the expected headers
	for name, values := range headers {
		// Loop over all values for the name.
		req.Header.Set(name, values)
	}

	// Set any query params
	q := req.URL.Query()
	for key, values := range params {
		q.Add(key, values)
	}

	// Add our clientProtocol param
	q.Add("clientProtocol", "1.5")
	req.URL.RawQuery = q.Encode()

	// Make the request and wait for the body to close
	log.Printf("Starting negotiation with URL %s", negotiateUrl)
	res, _ := httpClient.Do(req)
	defer res.Body.Close()

	// Extract out the connection token
	bodyBytes, _ := ioutil.ReadAll(res.Body)
	var m map[string]interface{}
	err := json.Unmarshal(bodyBytes, &m)
	if err != nil {
		// TODO: Add error handling around this, we should at least retry and then bubble up the error to the user
		log.Println("Error un-marshalling response! TODO Fix me")
		panic(err)
	}
	connectionId := m["connectionId"]

	// Add the connection id to the list of params
	params["id"] = connectionId.(string)
	params["clientProtocol"] = "1.5"
	params["transport"] = "WebSockets"

	// Make an interrupt channel
	// TODO: Think this can be removed
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Build our url u , add our params as well
	u := url.URL{Scheme: "wss", Host: serviceUrl, Path: hubEndpoint}
	q = u.Query()
	for key, value := range params {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()

	log.Printf("Negotiation finished, received %d. Connecting to %s", res.StatusCode, u.String())

	// Connect to the websocket, catch any errors
	// TODO: Get ride of this header req
	ret.Client, _, err = websocket.DefaultDialer.Dial(u.String(), http.Header{"Authorization": []string{headers["Authorization"]}})
	if err != nil {
		log.Fatal("dial:", err)
	}

	// Make our response channel
	ret.WebsocketMessageChan = make(chan []byte)

	// Define our protocol and version
	// Ref: https://stackoverflow.com/questions/65214787/signalr-websockets-and-go
	if err = ret.Client.WriteMessage(websocket.TextMessage, append([]byte(`{"protocol": "json","version": 1}`), 0x1E)); err != nil {
		return nil
	}

	// Make a done channel - not really sure what this does
	done := make(chan struct{})

	// Set up our listener to alert on the channel when we get a message
	go func() {
		defer close(done)
		for {
			// Keep reading messages that come in
			_, message, err := ret.Client.ReadMessage()
			if err != nil {
				// TODO: Handle this error better
				log.Println("ERROR IN WEBSOCKET MESSAGE: ", err)
				return
			}
			// Always trim off the termination char if its there
			if message[len(message)-1] == messageTerminator {
				message = message[0 : len(message)-1]
			}

			// Also check to see if we have multiple messages
			seporatedMessages := bytes.Split(message, []byte{messageTerminator})

			for _, formattedMessage := range seporatedMessages {
				// And alert on our channel
				ret.WebsocketMessageChan <- formattedMessage
			}
		}
	}()
	return &ret
}

// type UniqueRand struct {
// 	generated map[int]bool
// }

// func NewWebsocketClient(authHeader string, sessionId string, assumeRole string, serviceURL string, clientIdentifier string) *WebsocketClient {

// 	// Make our headers
// 	headers := make(map[string]string)
// 	headers["Authorization"] = authHeader

// 	// Make our params
// 	params := make(map[string]string)
// 	params["session_id"] = sessionId

// 	// If we are the client, pass the assume_role info as well to the params
// 	if assumeRole != "" {
// 		params["assume_role"] = assumeRole
// 		ret.IsServer = false
// 	}

// 	// If we are the server, pass the clientIdentifier info to the params
// 	if clientIdentifier != "" {
// 		params["client_identifier"] = clientIdentifier
// 		ret.IsServer = true

// 		// Servers are always ready as they start the connnection
// 		ret.IsReady = true
// 	}

// 	// First negotiate in order to get a url to connect to
// 	httpClient := &http.Client{}
// 	negotiateUrl := "https://" + serviceURL + "/api/v1/hub/kube/negotiate"
// 	req, _ := http.NewRequest("POST", negotiateUrl, nil)

// 	// Add the expected headers
// 	for name, values := range headers {
// 		// Loop over all values for the name.
// 		req.Header.Set(name, values)
// 	}

// 	// Set any query params
// 	q := req.URL.Query()
// 	for key, values := range params {
// 		q.Add(key, values)
// 	}

// 	// Add our clientProtocol param
// 	q.Add("clientProtocol", "1.5")
// 	req.URL.RawQuery = q.Encode()

// 	// Make the request and wait for the body to close
// 	log.Printf("Starting negotiation with URL %s", negotiateUrl)
// 	res, _ := httpClient.Do(req)
// 	defer res.Body.Close()

// 	// Extract out the connection token
// 	bodyBytes, _ := ioutil.ReadAll(res.Body)
// 	var m map[string]interface{}
// 	err := json.Unmarshal(bodyBytes, &m)
// 	if err != nil {
// 		// TODO: Add error handling around this, we should at least retry and then bubble up the error to the user
// 		panic(err)
// 	}
// 	connectionId := m["connectionId"]

// 	// Add the connection id to the list of params
// 	params["id"] = connectionId.(string)
// 	params["clientProtocol"] = "1.5"
// 	params["transport"] = "WebSockets"

// 	// Make an interrupt channel
// 	interrupt := make(chan os.Signal, 1)
// 	signal.Notify(interrupt, os.Interrupt)

// 	// Build our url u , add our params as well
// 	u := url.URL{Scheme: "wss", Host: serviceURL, Path: "/api/v1/hub/kube"}
// 	q = u.Query()
// 	for key, value := range params {
// 		q.Set(key, value)
// 	}
// 	u.RawQuery = q.Encode()

// 	log.Printf("Negotiation finished, received %d. Connecting to %s", res.StatusCode, u.String())

// 	// Connect to the websocket, catch any errors
// 	ret.Client, _, err = websocket.DefaultDialer.Dial(u.String(), http.Header{"Authorization": []string{authHeader}})
// 	if err != nil {
// 		log.Fatal("dial:", err)
// 	}
// 	// Save the client in the object
// 	ret.SignalRTypeNumber = 1

// 	// Add our response channels
// 	ret.DataToClientChan = make(chan websocketClientTypes.DataToClientMessage)
// 	ret.RequestForServerChan = make(chan websocketClientTypes.RequestForServerSignalRMessage)
// 	ret.RequestForStartExecChan = make(chan websocketClientTypes.RequestForStartExecToClusterSingalRMessage)
// 	ret.ExecStdoutChannel = make(chan websocketClientTypes.SendStdoutToDaemonSignalRMessage)
// 	ret.ExecStdinChannel = make(chan websocketClientTypes.SendStdinToClusterSignalRMessage)

// 	// Define our protocol and version
// 	// Ref: https://stackoverflow.com/questions/65214787/signalr-websockets-and-go
// 	if err = ret.Client.WriteMessage(websocket.TextMessage, append([]byte(`{"protocol": "json","version": 1}`), 0x1E)); err != nil {
// 		return nil
// 	}

// 	// Make a done channel
// 	done := make(chan struct{})

// 	// Subscribe to our streams
// 	go func() {
// 		defer close(done)
// 		for {

// 			_, message, err := ret.Client.ReadMessage()
// 			if err != nil {
// 				log.Println("ERROR: ", err)
// 				return
// 			}

// 			// Always trim off the termination char if its there
// 			if message[len(message)-1] == messageTerminator {
// 				message = message[0 : len(message)-1]
// 			}

// 			// Also check to see if we have multiple messages
// 			seporatedMessages := bytes.Split(message, []byte{messageTerminator})

// 			for _, formattedMessage := range seporatedMessages {
// 				// Route to our handlers based on their target
// 				if bytes.Contains(formattedMessage, []byte("\"target\":\"DataToClient\"")) {
// 					log.Printf("Handling incoming DataToClient message")
// 					dataToClientSignalRMessage := new(websocketClientTypes.DataToClientSignalRMessage)
// 					err := json.Unmarshal(formattedMessage, dataToClientSignalRMessage)
// 					if err != nil {
// 						log.Printf("Error un-marshalling DataToClientSignalRMessage: %s", err)
// 						return
// 					}

// 					// Broadcase this response to our DataToClientChan
// 					ret.DataToClientChan <- dataToClientSignalRMessage.Arguments[0]
// 				} else if bytes.Contains(formattedMessage, []byte("\"target\":\"ReadyToClient\"")) {
// 					log.Printf("Handling incoming ReadyToClient message")
// 					readyFromServerSignalRMessage := new(websocketClientTypes.ReadyFromServerSignalRMessage)
// 					err := json.Unmarshal(formattedMessage, readyFromServerSignalRMessage)
// 					if err != nil {
// 						log.Printf("Error un-marshalling ReadyFromServerSignalRMessage: %s", err)
// 						return
// 					}
// 					if readyFromServerSignalRMessage.Arguments[0].Ready == true {
// 						ret.IsReady = true
// 					}
// 				} else if bytes.Contains(formattedMessage, []byte("\"target\":\"RequestForServer\"")) {
// 					log.Printf("Handling incoming RequestForServer message")
// 					requestForServerSignalRMessage := new(websocketClientTypes.RequestForServerSignalRMessage)

// 					err := json.Unmarshal(formattedMessage, requestForServerSignalRMessage)
// 					if err != nil {
// 						log.Printf("Error un-marshalling RequestForServerSignalRMessage: %s", err)
// 						return
// 					}
// 					// Broadcase this response to our DataToClientChan
// 					log.Printf("REQ IDENT: %d", requestForServerSignalRMessage.Arguments[0].RequestIdentifier)
// 					ret.RequestForServerChan <- *requestForServerSignalRMessage
// 				} else if bytes.Contains(formattedMessage, []byte("\"target\":\"StartExecToCluster\"")) {
// 					log.Printf("Handling incoming StartExecToCluster message")
// 					requestForStartExecToClusterSingalRMessage := new(websocketClientTypes.RequestForStartExecToClusterSingalRMessage)

// 					err := json.Unmarshal(formattedMessage, requestForStartExecToClusterSingalRMessage)
// 					if err != nil {
// 						log.Printf("Error un-marshalling StartExecToCluster: %s", err)
// 						return
// 					}
// 					// Broadcase this response to our RequestForStartExecChan
// 					ret.RequestForStartExecChan <- *requestForStartExecToClusterSingalRMessage
// 				} else if bytes.Contains(formattedMessage, []byte("\"target\":\"SendStdoutToDaemon\"")) {
// 					log.Printf("Handling incoming SendStdoutToDaemon message")
// 					sendStdoutToDaemonSignalRMessage := new(websocketClientTypes.SendStdoutToDaemonSignalRMessage)

// 					err := json.Unmarshal(formattedMessage, sendStdoutToDaemonSignalRMessage)
// 					if err != nil {
// 						log.Printf("Error un-marshalling SendStdoutToDaemon: %s", err)
// 						return
// 					}
// 					// Broadcase this response to our RequestForStartExecChan
// 					ret.ExecStdoutChannel <- *sendStdoutToDaemonSignalRMessage
// 				} else if bytes.Contains(formattedMessage, []byte("\"target\":\"SendStdinToCluster\"")) {
// 					log.Printf("Handling incoming SendStdinToCluster message")
// 					sendStdinToClusterSignalRMessage := new(websocketClientTypes.SendStdinToClusterSignalRMessage)

// 					err := json.Unmarshal(formattedMessage, sendStdinToClusterSignalRMessage)
// 					if err != nil {
// 						log.Printf("Error un-marshalling SendStdoutToDaemon: %s", err)
// 						return
// 					}
// 					// Broadcase this response to our RequestForStartExecChan
// 					ret.ExecStdinChannel <- *sendStdinToClusterSignalRMessage
// 				} else {
// 					log.Printf("Unhandled message incoming: %s", formattedMessage)
// 				}
// 			}
// 		}
// 	}()
// 	return &ret
// }

// // Function to send data Bastion from a DataFromClientMessage object
// func (client *WebsocketClient) SendDataFromClientMessage(dataFromClientMessage websocketClientTypes.DataFromClientMessage) error {
// 	if !client.IsServer && client.IsReady {
// 		// Lock our mutex and setup the unlock
// 		client.SocketLock.Lock()
// 		defer client.SocketLock.Unlock()

// 		log.Printf("Sending data to Bastion")

// 		// Create the object, add relevent information
// 		toSend := new(websocketClientTypes.DataFromClientSignalRMessage)
// 		toSend.Target = "DataFromClient"
// 		toSend.Arguments = []websocketClientTypes.DataFromClientMessage{dataFromClientMessage}

// 		// Add the type number from the class
// 		toSend.Type = 1 // Ref: https://github.com/aspnet/SignalR/blob/master/specs/HubProtocol.md#invocation-message-encoding

// 		// Marshal our message
// 		toSendMarshalled, err := json.Marshal(toSend)
// 		if err != nil {
// 			return err
// 		}

// 		// Write our message
// 		if err = client.Client.WriteMessage(websocket.TextMessage, append(toSendMarshalled, 0x1E)); err != nil {
// 			return err
// 		}
// 		// client.SignalRTypeNumber++
// 		return nil
// 	}
// 	// TODO: Return error
// 	return nil
// }

// func (client *WebsocketClient) SendResponseToDaemonMessage(responseToDaemonMessage websocketClientTypes.ResponseToDaemonMessage) error {
// 	if client.IsServer && client.IsReady {
// 		// Lock our mutex and setup the unlock
// 		client.SocketLock.Lock()
// 		defer client.SocketLock.Unlock()

// 		log.Printf("Sending data to Daemon")
// 		// Create the object, add relevent information
// 		toSend := new(websocketClientTypes.ResponseToDaemonSignalRMessage)
// 		toSend.Target = "ResponseToDaemon"
// 		toSend.Arguments = []websocketClientTypes.ResponseToDaemonMessage{responseToDaemonMessage}

// 		// Add the type number from the class
// 		toSend.Type = 1 // Ref: https://github.com/aspnet/SignalR/blob/master/specs/HubProtocol.md#invocation-message-encoding

// 		// Marshal our message
// 		toSendMarshalled, err := json.Marshal(toSend)
// 		if err != nil {
// 			return err
// 		}

// 		// Write our message
// 		if err = client.Client.WriteMessage(websocket.TextMessage, append(toSendMarshalled, 0x1E)); err != nil {
// 			return err
// 		}
// 		// client.SignalRTypeNumber++
// 		return nil
// 	}
// 	// TODO: Return error
// 	return nil
// }

// func (client *WebsocketClient) SendStartExecToBastionMessage(startExecToBastionMessage websocketClientTypes.StartExecToBastionMessage) error {
// 	if !client.IsServer && client.IsReady {
// 		// Lock our mutex and setup the unlock
// 		client.SocketLock.Lock()
// 		defer client.SocketLock.Unlock()

// 		log.Printf("Sending data to Cluster")
// 		// Create the object, add relevent information
// 		toSend := new(websocketClientTypes.StartExecToBastionSignalRMessage)
// 		toSend.Target = "StartExecToBastion"
// 		toSend.Arguments = []websocketClientTypes.StartExecToBastionMessage{startExecToBastionMessage}

// 		// Add the type number from the class
// 		toSend.Type = 1 // Ref: https://github.com/aspnet/SignalR/blob/master/specs/HubProtocol.md#invocation-message-encoding

// 		// Marshal our message
// 		toSendMarshalled, err := json.Marshal(toSend)
// 		if err != nil {
// 			return err
// 		}

// 		// Write our message
// 		if err = client.Client.WriteMessage(websocket.TextMessage, append(toSendMarshalled, 0x1E)); err != nil {
// 			log.Printf("Something went wrong :(")
// 			return err
// 		}
// 		// client.SignalRTypeNumber++
// 		return nil
// 	}
// 	// TODO: Return error
// 	return nil
// }

// func (client *WebsocketClient) SendSendStdoutToBastionMessage(sendStdoutToBastionMessage websocketClientTypes.SendStdoutToBastionMessage) error {
// 	if client.IsServer && client.IsReady {
// 		// Lock our mutex and setup the unlock
// 		client.SocketLock.Lock()
// 		defer client.SocketLock.Unlock()

// 		log.Printf("Sending stdout to Cluster")
// 		// Create the object, add relevent information
// 		toSend := new(websocketClientTypes.SendStdoutToBastionSignalRMessage)
// 		toSend.Target = "SendStdoutToBastion"
// 		toSend.Arguments = []websocketClientTypes.SendStdoutToBastionMessage{sendStdoutToBastionMessage}

// 		// Add the type number from the class
// 		toSend.Type = 1 // Ref: https://github.com/aspnet/SignalR/blob/master/specs/HubProtocol.md#invocation-message-encoding

// 		// Marshal our message
// 		toSendMarshalled, err := json.Marshal(toSend)
// 		if err != nil {
// 			return err
// 		}

// 		// Write our message
// 		if err = client.Client.WriteMessage(websocket.TextMessage, append(toSendMarshalled, 0x1E)); err != nil {
// 			log.Printf("Something went wrong :(")
// 			return err
// 		}
// 		// client.SignalRTypeNumber++
// 		return nil
// 	}
// 	// TODO: Return error
// 	return nil
// }

// func (client *WebsocketClient) SendSendStdinToBastionMessage(sendStdinToBastionMessage websocketClientTypes.SendStdinToBastionMessage) error {
// 	if !client.IsServer && client.IsReady {
// 		// Lock our mutex and setup the unlock
// 		client.SocketLock.Lock()
// 		defer client.SocketLock.Unlock()

// 		log.Printf("Sending stdout to Cluster")
// 		// Create the object, add relevent information
// 		toSend := new(websocketClientTypes.SendStdinToBastionSignalRMessage)
// 		toSend.Target = "SendStdinToBastion"
// 		toSend.Arguments = []websocketClientTypes.SendStdinToBastionMessage{sendStdinToBastionMessage}

// 		// Add the type number from the class
// 		toSend.Type = 1 // Ref: https://github.com/aspnet/SignalR/blob/master/specs/HubProtocol.md#invocation-message-encoding

// 		// Marshal our message
// 		toSendMarshalled, err := json.Marshal(toSend)
// 		if err != nil {
// 			return err
// 		}

// 		// Write our message
// 		if err = client.Client.WriteMessage(websocket.TextMessage, append(toSendMarshalled, 0x1E)); err != nil {
// 			log.Printf("Something went wrong :(")
// 			return err
// 		}
// 		// client.SignalRTypeNumber++
// 		return nil
// 	}
// 	// TODO: Return error
// 	return nil
// }

// // Helper function to generate a random unique identifier
// func (c *WebsocketClient) GenerateUniqueIdentifier() int {
// 	for {
// 		i := rand.Intn(10000)
// 		return i
// 		// TODO: Implement a unique check
// 		// if !u.generated[i] {
// 		// 	u.generated[i] = true
// 		// 	return i
// 		// }
// 	}
// }
