package commonWebsocketClient

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// This will be the client that we use to store our websocket connection
type WebsocketClient struct {
	Client            *websocket.Conn
	IsReady           bool
	SignalRTypeNumber int

	// Ref: https://github.com/gorilla/websocket/issues/119#issuecomment-198710015
	SocketLock sync.Mutex

	// These objects are used for closing the websocket
	Cancel context.CancelFunc
	Close  bool

	// This will be our one response channel whenever we get a websocket message
	WebsocketMessageChan chan []byte
}

// All SignalR Messages are teminated with this byte
const messageTerminator byte = 0x1E
const sleepIntervalInSeconds = 5

// Constructor to create a new common websocket client object that can be shared by the daemon and server
func NewCommonWebsocketClient(serviceUrl string, hubEndpoint string, params map[string]string, headers map[string]string) *WebsocketClient {

	ret := WebsocketClient{}

	// Connect to the websocket, catch any errors
	ret.ConnectToWebsocket(serviceUrl, hubEndpoint, headers, params)

	// Make our response channel
	ret.WebsocketMessageChan = make(chan []byte)

	// Make our cancel context
	ctx, cancel := context.WithCancel(context.Background())
	ret.Cancel = cancel
	ret.Close = false

	// Set up our listener to alert on the channel when we get a message
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Keep reading messages that come in
				_, message, err := ret.Client.ReadMessage()
				if err != nil && ret.Close == false {
					// If we see an error, and we are not trying to close the connection
					// TODO: Handle this error better
					log.Println("Error in websocket, will attempt to reconnect: ", err)
					ret.IsReady = false
					ret.ConnectToWebsocket(serviceUrl, hubEndpoint, headers, params)
				} else if ret.Close == true {
					// If we are trying to close to connection, end the goroutine
					return
				} else {
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
				break
			}
		}
	}()
	return &ret
}

func (wsClient *WebsocketClient) ConnectToWebsocket(serviceUrl string, hubEndpoint string, headers map[string]string, params map[string]string) {
	connected := false
	for connected == false {

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
			log.Printf("Error un-marshalling negotiate response: %s", m)
			connected = false
		} else {
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
			websocketUrl := url.URL{Scheme: "wss", Host: serviceUrl, Path: hubEndpoint}
			q = websocketUrl.Query()
			for key, value := range params {
				q.Set(key, value)
			}
			websocketUrl.RawQuery = q.Encode()

			log.Printf("Negotiation finished, received %d. Connecting to %s", res.StatusCode, websocketUrl.String())

			wsClient.Client, _, err = websocket.DefaultDialer.Dial(websocketUrl.String(), http.Header{"Authorization": []string{headers["Authorization"]}})

			// Define our protocol and version
			// Ref: https://stackoverflow.com/questions/65214787/signalr-websockets-and-go
			if err := wsClient.Client.WriteMessage(websocket.TextMessage, append([]byte(`{"protocol": "json","version": 1}`), 0x1E)); err != nil {
				log.Println("Error when trying to agree on version for SignalR!")
				connected = false
				wsClient.Client.Close()
			}
		}

		if err != nil {
			connected = false
		} else {
			connected = true
			wsClient.IsReady = true
			break
		}

		// Sleep in between
		log.Printf("Connecting failed! Sleeping for %d seconds before attempting again", sleepIntervalInSeconds)
		time.Sleep(time.Second * sleepIntervalInSeconds)
	}
}
