package websocket

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	"bastionzero.com/bctl/v1/bctl/agent/vault"
	wsmsg "bastionzero.com/bctl/v1/bzerolib/channels/message"
	lggr "bastionzero.com/bctl/v1/bzerolib/logger"

	"github.com/gorilla/websocket"
)

const (
	sleepIntervalInSeconds = 5
	connectionTimeout      = 30 // just a reminder for now

	challengeEndpoint = "/api/v1/kube/get-challenge"

	// SignalR
	signalRMessageTerminatorByte = 0x1E
	signalRTypeNumber            = 1 // Ref: https://github.com/aspnet/SignalR/blob/master/specs/HubProtocol.md#invocation-message-encoding

	cleanWebsocketExit = "websocket: close 1000 (normal)"
)

type IWebsocket interface {
	Connect() error
	Receive()
	Send(agentMessage wsmsg.AgentMessage) error
}

// This will be the client that we use to store our websocket connection
type Websocket struct {
	Client  *websocket.Conn
	logger  *lggr.Logger
	IsReady bool

	// Ref: https://github.com/gorilla/websocket/issues/119#issuecomment-198710015
	SocketLock sync.Mutex

	// These are the channels for recieving and sending messages and done
	InputChannel  chan wsmsg.AgentMessage
	OutputChannel chan wsmsg.AgentMessage
	DoneChannel   chan bool

	// Function for figuring out correct Target SignalR Hub
	targetSelectHandler func(msg wsmsg.AgentMessage) (string, error)

	// Flag to indicate if we should automatically try to reconnect
	autoReconnect bool

	getChallenge bool

	// Connection variables
	serviceUrl  string
	hubEndpoint string
	params      map[string]string
	headers     map[string]string
}

// Constructor to create a new common websocket client object that can be shared by the daemon and server
func NewWebsocket(logger *lggr.Logger,
	serviceUrl string,
	hubEndpoint string,
	params map[string]string,
	headers map[string]string,
	targetSelectHandler func(msg wsmsg.AgentMessage) (string, error),
	autoReconnect bool,
	getChallenge bool) (*Websocket, error) {

	ctx := context.TODO() // TODO: get this from parent channel

	ret := Websocket{
		logger:              logger,
		InputChannel:        make(chan wsmsg.AgentMessage, 200),
		OutputChannel:       make(chan wsmsg.AgentMessage, 200),
		DoneChannel:         make(chan bool),
		targetSelectHandler: targetSelectHandler,
		getChallenge:        getChallenge,
		autoReconnect:       autoReconnect,
		serviceUrl:          serviceUrl,
		hubEndpoint:         hubEndpoint,
		params:              params,
		headers:             headers,
	}

	ret.Connect()

	// Listener for any outgoing messages
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
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
				if err := ret.Receive(); err != nil {
					ret.logger.Error(err)
					ret.DoneChannel <- true
					return
				}
			}
		}
	}()
	return &ret, nil
}

// Returns error on websocket closed
func (w *Websocket) Receive() error {
	// Read incoming message(s)
	_, rawMessage, err := w.Client.ReadMessage()

	if err != nil {
		w.IsReady = false

		// Check if it's a clean exit or we don't need to reconnect
		if err.Error() == cleanWebsocketExit || !w.autoReconnect {
			return errors.New("Websocket closed")
		} else { // else, reconnect
			msg := fmt.Errorf("error in websocket, will attempt to reconnect: %s", err)
			w.logger.Error(msg)
			w.Connect()
		}
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
				msg := fmt.Errorf("error unmarshalling SignalR message from Bastion: %v", string(msg))
				w.logger.Error(msg)
				break
			}

			// push to channel
			if wrappedMessage.Type != signalRTypeNumber {
				msg := fmt.Sprintf("Ignoring SignalR message with type %v", wrappedMessage.Type)
				w.logger.Debug(msg)
			} else if len(wrappedMessage.Arguments) != 0 {
				if wrappedMessage.Target == "CloseConnection" {
					return errors.New("closing message received; websocket closed")
				}
				w.InputChannel <- wrappedMessage.Arguments[0]
			}
		}
	}
	return nil
}

// Function to write signalr message to websocket
func (w *Websocket) Send(agentMessage wsmsg.AgentMessage) error {
	if !w.IsReady {
		return fmt.Errorf("Websocket not ready to send yet")
	}

	// Lock our mutex and setup the unlock
	w.SocketLock.Lock()
	defer w.SocketLock.Unlock()

	// Select target
	target, err := w.targetSelectHandler(agentMessage) // Agent and Daemon specify their own function to choose target
	if err != nil {
		return fmt.Errorf("error in selecting SignalR Endpoint target name: %s", err)
	}

	msg := fmt.Sprintf("Sending %s message to the Bastion", target)
	w.logger.Info(msg)

	signalRMessage := wsmsg.SignalRWrapper{
		Target:    target,
		Type:      signalRTypeNumber,
		Arguments: []wsmsg.AgentMessage{agentMessage},
	}

	if msgBytes, err := json.Marshal(signalRMessage); err != nil {
		return fmt.Errorf("error marshalling outgoing SignalR Message: %v", signalRMessage)
	} else {
		// Write our message to websocket
		if err = w.Client.WriteMessage(websocket.TextMessage, append(msgBytes, signalRMessageTerminatorByte)); err != nil {
			return err
		} else {
			return nil
		}
	}
}

func (w *Websocket) Connect() {
	for !w.IsReady {
		time.Sleep(time.Second * sleepIntervalInSeconds)
		if w.getChallenge {
			// First get the config from the vault
			config, _ := vault.LoadVault()

			// If we have a private key, we must solve the challenge
			solvedChallenge, err := newChallenge(w.params["org_id"], w.params["cluster_name"], w.serviceUrl, config.Data.PrivateKey)
			if err != nil {
				w.logger.Error(fmt.Errorf("error in getting challenge: %s", err))

				// Sleep in between
				w.logger.Info(fmt.Sprintf("Connecting failed! Sleeping for %d seconds before attempting again", sleepIntervalInSeconds))
				continue
			}

			// Add the solved challenge to the params
			w.params["solved_challenge"] = solvedChallenge
		}

		// First negotiate in order to get a url to connect to
		httpClient := &http.Client{}
		negotiateUrl := "https://" + w.serviceUrl + w.hubEndpoint + "/negotiate"
		req, _ := http.NewRequest("POST", negotiateUrl, nil)

		// Add the expected headers
		for name, values := range w.headers {
			// Loop over all values for the name.
			req.Header.Set(name, values)
		}

		// Set any query params
		q := req.URL.Query()
		for key, values := range w.params {
			q.Add(key, values)
		}

		// Add our clientProtocol param
		q.Add("clientProtocol", "1.5")
		req.URL.RawQuery = q.Encode()

		// Make the request and wait for the body to close
		w.logger.Info(fmt.Sprintf("Starting negotiation with URL %s", negotiateUrl))
		res, _ := httpClient.Do(req)
		defer res.Body.Close()

		if res.StatusCode == 401 {
			// This means we have an auth issue, do not attempt to keep trying to reconnect
			rerr := fmt.Errorf("Auth error when trying to connect. Not attempting to reconnect. Shutting down")
			w.logger.Error(rerr)
			return
		} else if res.StatusCode != 200 {
			w.logger.Error(fmt.Errorf("Bad status code received on negotiation: %s", res.StatusCode))

			// Sleep in between
			w.logger.Info(fmt.Sprintf("Connecting failed! Sleeping for %d seconds before attempting again", sleepIntervalInSeconds))
			continue
		}

		// Extract out the connection token
		bodyBytes, _ := ioutil.ReadAll(res.Body)
		var m map[string]interface{}

		if err := json.Unmarshal(bodyBytes, &m); err != nil {
			// TODO: Add error handling around this, we should at least retry and then bubble up the error to the user
			w.logger.Error(fmt.Errorf("error un-marshalling negotiate response: %s", m))
		}

		connectionId := m["connectionId"]

		// Add the connection id to the list of params
		w.params["id"] = connectionId.(string)
		w.params["clientProtocol"] = "1.5"
		w.params["transport"] = "WebSockets"

		// Make an interrupt channel
		// TODO: Think this can be removed
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)

		// Build our url u , add our params as well
		websocketUrl := url.URL{Scheme: "wss", Host: w.serviceUrl, Path: w.hubEndpoint}
		q = websocketUrl.Query()
		for key, value := range w.params {
			q.Set(key, value)
		}
		websocketUrl.RawQuery = q.Encode()

		msg := fmt.Sprintf("Negotiation finished, received %d. Connecting to %s", res.StatusCode, websocketUrl.String())
		w.logger.Info(msg)

		var err error
		w.Client, _, err = websocket.DefaultDialer.Dial(
			websocketUrl.String(),
			http.Header{"Authorization": []string{w.headers["Authorization"]}})
		if err != nil {
			w.logger.Error(err)
		} else {
			// Define our protocol and version
			// Ref: https://stackoverflow.com/questions/65214787/signalr-websockets-and-go
			if err := w.Client.WriteMessage(websocket.TextMessage, append([]byte(`{"protocol": "json","version": 1}`), signalRMessageTerminatorByte)); err != nil {
				w.logger.Info("Error when trying to agree on version for SignalR!")
				w.Client.Close()
			} else {
				w.IsReady = true
				break
			}
		}
	}
}
