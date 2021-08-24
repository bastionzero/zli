package websocket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	wsmsg "bastionzero.com/bctl/v1/bzerolib/channels/message"

	"github.com/gorilla/websocket"
)

const (
	sleepIntervalInSeconds = 5
	connectionTimeout      = 30 // just a reminder for now

	// SignalR
	signalRMessageTerminatorByte = 0x1E
	signalRTypeNumber            = 1 // Ref: https://github.com/aspnet/SignalR/blob/master/specs/HubProtocol.md#invocation-message-encoding
)

type IWebsocket interface {
	Connect(serviceUrl string, hubEndpoint string, headers map[string]string, params map[string]string) error
	Send(agentMessage wsmsg.AgentMessage) error
	// We should probably also have a close method here too
}

// This will be the client that we use to store our websocket connection
type Websocket struct {
	Client  *websocket.Conn
	IsReady bool

	// Ref: https://github.com/gorilla/websocket/issues/119#issuecomment-198710015
	SocketLock sync.Mutex

	// These objects are used for closing the websocket
	Cancel context.CancelFunc
	Closed bool

	// These are the channels for recieving and sending messages and done
	InputChannel  chan wsmsg.AgentMessage
	OutputChannel chan wsmsg.AgentMessage
	DoneChannel   chan string

	// Function for figuring out correct Target SignalR Hub
	targetSelectHandler func(msg wsmsg.AgentMessage) (string, error)

	// Flag to indicate if we should automatically try to reconnect
	autoReconnect bool
}

// Constructor to create a new common websocket client object that can be shared by the daemon and server
func NewWebsocket(serviceUrl string, hubEndpoint string, params map[string]string, headers map[string]string, targetSelectHandler func(msg wsmsg.AgentMessage) (string, error), autoReconnect bool) (*Websocket, error) {
	ctx, cancel := context.WithCancel(context.Background())

	ret := Websocket{
		Cancel: cancel,
		Closed: false,

		InputChannel:        make(chan wsmsg.AgentMessage, 200),
		OutputChannel:       make(chan wsmsg.AgentMessage, 200),
		DoneChannel:         make(chan string, 1),
		targetSelectHandler: targetSelectHandler,

		autoReconnect: autoReconnect,
	}

	// Connect to the websocket, catch any errors
	// if err := ret.Connect(serviceUrl, hubEndpoint, headers, params); err != nil {
	// 	return &ret, fmt.Errorf("Error connecting to websocket: %s", err.Error())
	// }

	ret.Connect(serviceUrl, hubEndpoint, headers, params)

	go func() {
		for {
			select {
			case msg := <-ret.OutputChannel:
				ret.Send(msg)
			}
		}
	}()

	// Set up our listener to alert on the channel when we get a message
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, rawMessage, err := ret.Client.ReadMessage()

				if err != nil && ret.Closed == false {
					// If we see an error, and we are not trying to close the connection
					// TODO: Handle this error better
					log.Println("Error in websocket, will attempt to reconnect: ", err)
					ret.IsReady = false
					if err.Error() == "websocket: close 1000 (normal)" {
						log.Println("Normal closure in websocket. Ending websocket connection")
						return
					}
					if ret.autoReconnect {
						ret.Connect(serviceUrl, hubEndpoint, headers, params)
					} else {
						log.Println("Auto-recoonect disabled, returning")
						return
					}
				} else if ret.Closed == true {
					// If we are trying to close to connection, end the goroutine
					return
				} else {
					// Always trim off the termination char if its there
					if rawMessage[len(rawMessage)-1] == signalRMessageTerminatorByte {
						rawMessage = rawMessage[0 : len(rawMessage)-1]
					}

					// Also check to see if we have multiple messages
					splitmessages := bytes.Split(rawMessage, []byte{signalRMessageTerminatorByte})

					for _, msg := range splitmessages {
						// unwrap signalR
						var wrappedMessage wsmsg.SignalRWrapper
						if err := json.Unmarshal(msg, &wrappedMessage); err != nil {
							log.Printf("Error unmarshalling SignalR message from Bastion: %v", string(msg))
							break
						}

						// push to channel
						if wrappedMessage.Type != signalRTypeNumber {
							log.Printf("Ignoring SignalR message with type %v", wrappedMessage.Type)
						} else if len(wrappedMessage.Arguments) != 0 {
							if wrappedMessage.Target == "CloseConnection" {
								log.Printf("Close Connection message received. Closing websocket")

								// Cancel our context
								cancel()

								// Deserialize the argument
								var closeMessage wsmsg.CloseMessage
								if err := json.Unmarshal(wrappedMessage.Arguments[0].MessagePayload, &closeMessage); err != nil {
									log.Printf("Error unmarshalling SignalR Close Message from Bastion: %v", string(wrappedMessage.Arguments[0].MessagePayload))
									break
								}

								// Send an alert on our done channel for our datachannel
								ret.DoneChannel <- closeMessage.Message

								return
							}
							ret.InputChannel <- wrappedMessage.Arguments[0]
						}
					}
				}
				break
			}
		}
	}()
	return &ret, nil
}

// Function to write signalr message to websocket
func (w *Websocket) Send(agentMessage wsmsg.AgentMessage) error {
	if !w.IsReady {
		return fmt.Errorf("Websocket not ready to send yet")
	}

	// Lock our mutex and setup the unlock
	w.SocketLock.Lock()
	defer w.SocketLock.Unlock()

	log.Printf("Sending message to the Bastion")

	// Select target
	target, err := w.targetSelectHandler(agentMessage)
	if err != nil {
		return fmt.Errorf("Error in selecting SignalR Endpoint target name: %v", err.Error())
	}
	log.Printf("Target: %v", target)

	signalRMessage := wsmsg.SignalRWrapper{
		Target:    target, // Leave up to daemon and agent to write more specific target specification function
		Type:      signalRTypeNumber,
		Arguments: []wsmsg.AgentMessage{agentMessage},
	}

	msgBytes, err := json.Marshal(signalRMessage)
	if err != nil {
		return fmt.Errorf("Error marshalling outgoing SignalR Message: %v", signalRMessage)
	}

	// Write our message to websocket
	if err = w.Client.WriteMessage(websocket.TextMessage, append(msgBytes, signalRMessageTerminatorByte)); err != nil {
		return err
	}

	return nil
}

// func (ws *Websocket) Connect(serviceUrl string, hubEndpoint string, headers map[string]string, params map[string]string) error {
// 	for ws.IsReady == false {

// 		// First negotiate in order to get a url to connect to
// 		httpClient := &http.Client{}
// 		negotiateUrl := "https://" + serviceUrl + hubEndpoint + "/negotiate"
// 		req, _ := http.NewRequest("POST", negotiateUrl, nil)

// 		// Add the expected headers
// 		for name, values := range headers {
// 			// Loop over all values for the name.
// 			req.Header.Set(name, values)
// 		}

// 		// Set any query params
// 		q := req.URL.Query()
// 		for key, values := range params {
// 			q.Add(key, values)
// 		}

// 		// Add our clientProtocol param
// 		q.Add("clientProtocol", "1.5")
// 		req.URL.RawQuery = q.Encode()

// 		// Make the request and wait for the body to close
// 		log.Printf("Starting negotiation with URL %s", negotiateUrl)
// 		res, _ := httpClient.Do(req)
// 		defer res.Body.Close()

// 		// Extract out the connection token
// 		bodyBytes, _ := ioutil.ReadAll(res.Body)
// 		var negotiateResp wsmsg.SignalRNegotiateResponse
// 		err := json.Unmarshal(bodyBytes, &negotiateResp)
// 		if err != nil {
// 			// TODO: Add error handling around this, we should at least retry and then bubble up the error to the user
// 			// TODO: Add status ok check
// 			return fmt.Errorf("Error un-marshalling negotiate response: %v", negotiateResp)
// 		}

// 		// Add the connection id to the list of params
// 		params["id"] = negotiateResp.ConnectionId
// 		params["clientProtocol"] = "1.5"
// 		params["transport"] = "WebSockets"

// 		// Make an interrupt channel
// 		// TODO: Think this can be removed
// 		interrupt := make(chan os.Signal, 1)
// 		signal.Notify(interrupt, os.Interrupt)

// 		// Build our url, add our params as well
// 		websocketUrl := url.URL{Scheme: "wss", Host: serviceUrl, Path: hubEndpoint}
// 		q = websocketUrl.Query()
// 		for key, value := range params {
// 			q.Set(key, value)
// 		}
// 		websocketUrl.RawQuery = q.Encode()

// 		log.Printf("Negotiation finished, received %d. Connecting to %s", res.StatusCode, websocketUrl.String())

// 		ws.Client, _, err = websocket.DefaultDialer.Dial(websocketUrl.String(), http.Header{"Authorization": []string{headers["Authorization"]}})

// 		// Define our protocol and version
// 		// Ref: https://stackoverflow.com/questions/65214787/signalr-websockets-and-go
// 		if err := ws.Client.WriteMessage(websocket.TextMessage, append([]byte(`{"protocol": "json","version": 1}`), 0x1E)); err != nil {
// 			log.Println("Error when trying to agree on version for SignalR!")
// 			ws.Client.Close()
// 		} else {
// 			ws.IsReady = true
// 			return nil
// 		}

// 		// Sleep in between
// 		log.Printf("Connecting failed! Sleeping for %d seconds before attempting again", sleepIntervalInSeconds)
// 		time.Sleep(time.Second * sleepIntervalInSeconds)
// 	}
// 	return nil
// }

func (w *Websocket) Connect(serviceUrl string, hubEndpoint string, headers map[string]string, params map[string]string) {
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

			w.Client, _, err = websocket.DefaultDialer.Dial(websocketUrl.String(), http.Header{"Authorization": []string{headers["Authorization"]}})

			// Define our protocol and version
			// Ref: https://stackoverflow.com/questions/65214787/signalr-websockets-and-go
			if err := w.Client.WriteMessage(websocket.TextMessage, append([]byte(`{"protocol": "json","version": 1}`), 0x1E)); err != nil {
				log.Println("Error when trying to agree on version for SignalR!")
				connected = false
				w.Client.Close()
			}
		}

		if err != nil {
			connected = false
		} else {
			connected = true
			w.IsReady = true
			break
		}

		// Sleep in between
		log.Printf("Connecting failed! Sleeping for %d seconds before attempting again", sleepIntervalInSeconds)
		time.Sleep(time.Second * sleepIntervalInSeconds)
	}
}
