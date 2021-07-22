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
				// TODO: This is where we need to try and reconnect
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
