package datachannel

import (
	"context"
	"encoding/json"
	"fmt"

	ks "bastionzero.com/bctl/v1/bctl/daemon/keysplitting"
	kube "bastionzero.com/bctl/v1/bctl/daemon/plugin/kube"
	wsmsg "bastionzero.com/bctl/v1/bzerolib/channels/message"
	ws "bastionzero.com/bctl/v1/bzerolib/channels/websocket"
	rrr "bastionzero.com/bctl/v1/bzerolib/error"
	ksmsg "bastionzero.com/bctl/v1/bzerolib/keysplitting/message"
	lggr "bastionzero.com/bctl/v1/bzerolib/logger"
	plgn "bastionzero.com/bctl/v1/bzerolib/plugin"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

const (
	maxRetries = 3
)

type IDataChannel interface {
	Send(messageType wsmsg.MessageType, messagePayload interface{}) error
	Receive(agentMessage wsmsg.AgentMessage) error
	StartKubeDaemonPlugin(localhostToken string, daemonPort string, certPath string, keyPath string) error
}

type DataChannel struct {
	websocket    *ws.Websocket
	logger       *lggr.Logger
	ctx          context.Context
	cancel       context.CancelFunc
	plugin       plgn.IPlugin
	keysplitting ks.IKeysplitting
	handshook    bool // aka whether we need to send a syn

	// Kube-specific vars aka to-be-removed
	role string

	// Done channel to bubble up messages to kubectl
	doneChannel chan string

	// If we need to send a SYN, then we need a way to keep
	// track of whatever message that triggered the send SYN
	onDeck      plgn.ActionWrapper
	lastMessage plgn.ActionWrapper
	retry       int
}

func NewDataChannel(logger *lggr.Logger,
	configPath string,
	role string,
	serviceUrl string,
	hubEndpoint string,
	params map[string]string,
	headers map[string]string,
	targetSelectHandler func(msg wsmsg.AgentMessage) (string, error),
	autoReconnect bool) (*DataChannel, error) {

	ctx, cancel := context.WithCancel(context.Background())

	subLogger := logger.GetWebsocketLogger()
	wsClient, err := ws.NewWebsocket(ctx, subLogger, serviceUrl, hubEndpoint, params, headers, targetSelectHandler, autoReconnect, false)
	if err != nil {
		cancel()
		logger.Error(err)
		return &DataChannel{}, err // TODO: how tf are we going to report these?
	}

	keysplitter, err := ks.NewKeysplitting("", configPath)
	if err != nil {
		cancel()
		logger.Error(err)
		return &DataChannel{}, err
	}

	ret := &DataChannel{
		websocket:    wsClient,
		logger:       logger,
		ctx:          ctx,
		cancel:       cancel,
		keysplitting: keysplitter,
		handshook:    false,
		doneChannel:  make(chan string),
		onDeck:       plgn.ActionWrapper{},
		retry:        0,
	}

	// Subscribe to our input channel
	go func() {
		for {
			select {
			case agentMessage := <-ret.websocket.InputChannel:
				// Handle each message in its own thread
				go func() {
					if err := ret.Receive(agentMessage); err != nil {
						ret.logger.Error(err)
					}
				}()
			case <-ret.websocket.DoneChannel:
				// The websocket has been closed
				msg := "Websocket has been closed, closing datachannel"
				ret.logger.Info(msg)
				cancel()

				// Send a message to our done channel so kubectl can display it
				ret.doneChannel <- msg
				return
			}
		}
	}()

	return ret, nil
}

func (d *DataChannel) StartKubeDaemonPlugin(localhostToken string, daemonPort string, certPath string, keyPath string) error {
	subLogger := d.logger.GetPluginLogger(plgn.KubeDaemon)
	if plugin, err := kube.NewKubeDaemonPlugin(d.ctx, subLogger, localhostToken, daemonPort, certPath, keyPath, d.doneChannel); err != nil {
		rerr := fmt.Errorf("could not start kube daemon plugin: %s", err)
		d.logger.Error(rerr)
		return rerr
	} else {
		d.plugin = plugin

		if err := d.sendSyn(); err != nil {
			return err
		}
		return nil
	}
}

// Wraps and sends the payload
func (d *DataChannel) Send(messageType wsmsg.MessageType, messagePayload interface{}) error {
	// Stop any further messages from being sent once context is cancelled
	if d.ctx.Err() == context.Canceled {
		return nil
	}

	messageBytes, _ := json.Marshal(messagePayload)
	agentMessage := wsmsg.AgentMessage{
		MessageType:    string(messageType),
		SchemaVersion:  wsmsg.SchemaVersion,
		MessagePayload: messageBytes,
	}

	// Push message to websocket channel output
	d.websocket.OutputChannel <- agentMessage
	return nil
}

func (d *DataChannel) sendSyn() error {
	d.logger.Info("Sending SYN")
	d.handshook = false
	payload := map[string]string{
		"Role": d.role,
	}
	payloadBytes, _ := json.Marshal(payload)

	action := "kube/restapi" // placeholder
	if synMessage, err := d.keysplitting.BuildSyn(action, payloadBytes); err != nil {
		rerr := fmt.Errorf("error building Syn: %s", err)
		d.logger.Error(rerr)
		return rerr
	} else {
		d.Send(wsmsg.Keysplitting, synMessage)
	}
	return nil
}

func (d *DataChannel) Receive(agentMessage wsmsg.AgentMessage) error {
	msg := fmt.Sprintf("Datachannel received %v message", wsmsg.MessageType(agentMessage.MessageType))
	d.logger.Info(msg)

	switch wsmsg.MessageType(agentMessage.MessageType) {
	case wsmsg.Keysplitting:
		var ksMessage ksmsg.KeysplittingMessage
		if err := json.Unmarshal(agentMessage.MessagePayload, &ksMessage); err != nil {
			rerr := fmt.Errorf("malformed Keysplitting message")
			d.logger.Error(rerr)
			return rerr
		} else {
			if err := d.handleKeysplittingMessage(&ksMessage); err != nil {
				d.logger.Error(err)
				return err
			}
		}
	case wsmsg.Stream:
		var sMessage smsg.StreamMessage
		if err := json.Unmarshal(agentMessage.MessagePayload, &sMessage); err != nil {
			rerr := fmt.Errorf("malformed Stream message")
			d.logger.Error(rerr)
			return rerr
		} else {
			if err := d.plugin.PushStreamInput(sMessage); err != nil {
				d.logger.Error(err)
				return err
			}
		}
	case wsmsg.Error:
		var errMessage rrr.ErrorMessage
		if err := json.Unmarshal(agentMessage.MessagePayload, &errMessage); err != nil {
			rerr := fmt.Errorf("malformed Error message")
			d.logger.Error(rerr)
			return rerr
		} else {
			rerr := fmt.Errorf("received error from agent: %s", errMessage.Message)
			d.logger.Error(rerr)

			// Keysplitting validation errors are probably going to be mostly bzcert renewals and
			// we don't want to break every time that happens so we need to get back on the ks train
			// executive decision: we don't retry if we get an error on a syn aka d.handshook == false
			if rrr.ErrorType(errMessage.Type) == rrr.KeysplittingValidationError && d.handshook {
				d.retry++
				d.onDeck = d.lastMessage

				// In order to get back on the keysplitting train, we need to resend the syn, get the synack
				// so that our input message handler is pointing to the right thing.
				if err := d.sendSyn(); err != nil {
					d.logger.Error(err)
					return err
				} else {
					return rerr
				}
			}

			d.doneChannel <- rerr.Error()
			d.cancel()
			return rerr
		}
	default:
		rerr := fmt.Errorf("unhandled Message type: %v", agentMessage.MessageType)
		d.logger.Error(rerr)
		return rerr
	}
	return nil
}

// TODO: simplify this and have them both deserialize into a "common keysplitting" message
func (d *DataChannel) handleKeysplittingMessage(keysplittingMessage *ksmsg.KeysplittingMessage) error {
	if err := d.keysplitting.Validate(keysplittingMessage); err != nil {
		rerr := fmt.Errorf("invalid keysplitting message: %s", err)
		d.logger.Error(rerr)
		return rerr
	}

	var action string
	var actionResponsePayload []byte
	switch keysplittingMessage.Type {
	case ksmsg.SynAck:
		synAckPayload := keysplittingMessage.KeysplittingPayload.(ksmsg.SynAckPayload)
		action = synAckPayload.Action
		actionResponsePayload = synAckPayload.ActionResponsePayload

		d.handshook = true

		// If there is a message that wasn't sent because we got a keysplitting validation error on it, send it now
		if d.onDeck.Action != "" {
			err := d.sendKeysplittingMessage(keysplittingMessage, d.onDeck.Action, d.onDeck.ActionPayload)
			return err
		}
	case ksmsg.DataAck:
		// If we had something on deck, then this was the ack for it and we can remove it
		d.onDeck = plgn.ActionWrapper{}
		// If we're here, it means that the previous data message that caused the error was accepted
		d.retry = 0

		dataAckPayload := keysplittingMessage.KeysplittingPayload.(ksmsg.DataAckPayload)
		action = dataAckPayload.Action
		actionResponsePayload = dataAckPayload.ActionResponsePayload
	default:
		rerr := fmt.Errorf("unhandled Keysplitting type")
		d.logger.Error(rerr)
		return rerr
	}

	// Send message to plugin's input message handler
	if action, returnPayload, err := d.plugin.InputMessageHandler(action, actionResponsePayload); err == nil {

		// We need to know the last message for invisible response to keysplitting validation errors
		d.lastMessage = plgn.ActionWrapper{
			Action:        action,
			ActionPayload: returnPayload,
		}

		return d.sendKeysplittingMessage(keysplittingMessage, action, returnPayload)

	} else {
		d.logger.Error(err)
		return err
	}
}

func (d *DataChannel) sendKeysplittingMessage(keysplittingMessage *ksmsg.KeysplittingMessage, action string, payload []byte) error {
	// Build and send response
	if respKSMessage, err := d.keysplitting.BuildResponse(keysplittingMessage, action, payload); err != nil {
		rerr := fmt.Errorf("could not build response message: %s", err)
		d.logger.Error(rerr)
		return rerr
	} else {
		d.Send(wsmsg.Keysplitting, respKSMessage)
		return nil
	}
}
