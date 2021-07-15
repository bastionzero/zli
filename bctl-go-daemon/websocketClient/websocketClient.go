package websocketClient

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"

	"bastionzero.com/bctl-daemon/v1/websocketClient/websocketClientTypes"
	"github.com/gorilla/websocket"
)

// This will be the client that we use to store our websocket connection
type WebsocketClient struct {
	Client            *websocket.Conn
	SignalRTypeNumber int
	DataToClientQueue map[int]websocketClientTypes.DataToClientMessage
	DataToClientChan  chan websocketClientTypes.DataToClientMessage
}

type UniqueRand struct {
	generated map[int]bool
}

func NewWebsocketClient(authHeader string, sessionId string, assumeRole string, serviceURL string) *WebsocketClient {
	// Constructor to create a new websocket client object
	ret := WebsocketClient{}

	// Make our headers
	headers := make(map[string]string)
	headers["Authorization"] = authHeader

	// Make our params
	params := make(map[string]string)
	params["session_id"] = sessionId
	params["assume_role"] = assumeRole

	// First negotiate in order to get a url to connect to
	httpClient := &http.Client{}
	negotiateUrl := "https://" + serviceURL + "/api/v1/hub/kube/negotiate"
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
		panic(err)
	}
	connectionId := m["connectionId"]

	// Add the connection id to the list of params
	params["id"] = connectionId.(string)
	params["clientProtocol"] = "1.5"
	params["transport"] = "WebSockets"

	// Make an interrupt channel
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Build our url u , add our params as well
	u := url.URL{Scheme: "wss", Host: serviceURL, Path: "/api/v1/hub/kube"}
	q = u.Query()
	for key, value := range params {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()

	log.Printf("Negotiation finished, received %d. Connecting to %s", res.StatusCode, u.String())

	// Connect to the websocket, catch any errors
	ret.Client, _, err = websocket.DefaultDialer.Dial(u.String(), http.Header{"Authorization": []string{authHeader}})
	if err != nil {
		log.Fatal("dial:", err)
	}
	// Save the client in the object
	ret.SignalRTypeNumber = 1

	// Add our response channels
	ret.DataToClientChan = make(chan websocketClientTypes.DataToClientMessage)

	// Define our protocol and version
	// Ref: https://stackoverflow.com/questions/65214787/signalr-websockets-and-go
	if err = ret.Client.WriteMessage(websocket.TextMessage, append([]byte(`{"protocol": "json","version": 1}`), 0x1E)); err != nil {
		return nil
	}

	// Make a done channel
	done := make(chan struct{})

	// Subscribe to our streams
	go func() {
		defer close(done)
		for {
			_, message, err := ret.Client.ReadMessage()
			if err != nil {
				log.Println("ERROR: ", err)
				return
			}
			// Always trim off the termination char
			message = bytes.Trim(message, "\x1e")

			// Route to our handlers based on their target
			if bytes.Contains(message, []byte("\"target\":\"DataToClient\"")) {
				log.Printf("Handling incoming DataToClient message")
				dataToClientSignalRMessage := new(websocketClientTypes.DataToClientSignalRMessage)
				err := json.Unmarshal(message, &dataToClientSignalRMessage)
				if err != nil {
					log.Printf("Error un-marshalling DataToClientSignalRMessage: %s", err)
					log.Printf(string(message))
				}

				// Broadcase this response to our DataToClientChan
				ret.DataToClientChan <- dataToClientSignalRMessage.Arguments[0]
			} else {
				log.Printf("Unhandled message incoming: %s", message)
			}
		}
	}()

	// // Imagine some incoming message coming in
	// dataFromClientMessage := new(DataFromClientMessage)
	// headr := new(Headers)
	// dataFromClientMessage.LogId = "e80d6510-fb36-4de1-9478-397d80ac43d8"
	// dataFromClientMessage.KubeCommand = "bctl test"
	// dataFromClientMessage.Endpoint = "/test"
	// dataFromClientMessage.Headers = *headr
	// dataFromClientMessage.Method = "Get"
	// dataFromClientMessage.Body = "test"
	// dataFromClientMessage.RequestIdentifier = 1

	// // First send that message to Bastion
	// ret.SendDataFromClientMessage(*dataFromClientMessage)

	// // Wait for the response
	// // dataToClientMessage := new(DataToClientMessage)
	// dataToClientMessage := <-ret.DataToClientChan

	return &ret
}

// Function to send data Bastion from a DataFromClientMessage object
func (client WebsocketClient) SendDataFromClientMessage(dataFromClientMessage websocketClientTypes.DataFromClientMessage) error {
	log.Printf("Sending data to Bastion")

	// Create the object, add relevent information
	toSend := new(websocketClientTypes.DataFromClientSignalRMessage)
	toSend.Target = "DataFromClient"
	toSend.Arguments = []websocketClientTypes.DataFromClientMessage{dataFromClientMessage}

	// Add the type number from the class
	toSend.Type = client.SignalRTypeNumber

	// Marshal our message
	toSendMarshalled, err := json.Marshal(toSend)
	if err != nil {
		return err
	}

	// Write our message
	if err = client.Client.WriteMessage(websocket.TextMessage, append(toSendMarshalled, 0x1E)); err != nil {
		return err
	}
	client.SignalRTypeNumber++
	return nil
}

// Helper function to generate a random unique identifier
func (c *WebsocketClient) GenerateUniqueIdentifier() int {
	for {
		i := rand.Intn(10000)
		return i
		// TODO: Implement a unique check
		// if !u.generated[i] {
		// 	u.generated[i] = true
		// 	return i
		// }
	}
}
