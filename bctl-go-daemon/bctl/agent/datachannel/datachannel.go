package datachannel

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	ks "bastionzero.com/bctl/v1/bctl/agent/keysplitting"
	kube "bastionzero.com/bctl/v1/bctl/agent/plugin/kube"
	wsmsg "bastionzero.com/bctl/v1/bzerolib/channels/message"
	ws "bastionzero.com/bctl/v1/bzerolib/channels/websocket"
	ksmsg "bastionzero.com/bctl/v1/bzerolib/keysplitting/message"
	lggr "bastionzero.com/bctl/v1/bzerolib/logger"
	plgn "bastionzero.com/bctl/v1/bzerolib/plugin"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

type IDataChannel interface {
	Send(messageType wsmsg.MessageType, messagePayload interface{}) error
	Receive(agentMessage wsmsg.AgentMessage) error
}

type DataChannel struct {
	websocket *ws.Websocket
	logger    *lggr.Logger
	ctx       context.Context

	plugin       plgn.IPlugin
	keysplitting ks.IKeysplitting

	// Kube-specific vars
	role string
}

func NewDataChannel(logger *lggr.Logger,
	role string,
	serviceUrl string,
	hubEndpoint string,
	params map[string]string,
	headers map[string]string,
	targetSelectHandler func(msg wsmsg.AgentMessage) (string, error),
	autoReconnect bool) (*DataChannel, error) {
	subLogger := logger.GetWebsocketLogger()

	wsClient, err := ws.NewWebsocket(subLogger, serviceUrl, hubEndpoint, params, headers, targetSelectHandler, autoReconnect, false)
	if err != nil {
		logger.Error(err)
		return &DataChannel{}, err // TODO: how tf are we going to report these? control channel, bro
	}

	keysplitter, err := ks.NewKeysplitting()
	if err != nil {
		logger.Error(err)
		return &DataChannel{}, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	ret := &DataChannel{
		websocket:    wsClient,
		keysplitting: keysplitter,
		role:         role,
		logger:       logger, // TODO: get debug level from flag
		ctx:          ctx,
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
				ret.logger.Info("Websocket has been closed, closing datachannel")
				cancel()
				return
			}
		}
	}()

	return ret, nil
}

// Wraps and sends the payload
func (d *DataChannel) Send(messageType wsmsg.MessageType, messagePayload interface{}) error {
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

func (d *DataChannel) Receive(agentMessage wsmsg.AgentMessage) error {
	msg := fmt.Sprintf("received %v message", wsmsg.MessageType(agentMessage.MessageType))
	d.logger.Info(msg)

	switch wsmsg.MessageType(agentMessage.MessageType) {
	case wsmsg.Keysplitting:
		var ksMessage ksmsg.KeysplittingMessage
		if err := json.Unmarshal(agentMessage.MessagePayload, &ksMessage); err != nil {
			return fmt.Errorf("malformed Keysplitting message")
		} else {
			if err := d.handleKeysplittingMessage(&ksMessage); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unhandled message type: %v", agentMessage.MessageType)
	}
	return nil
}

func (d *DataChannel) handleKeysplittingMessage(keysplittingMessage *ksmsg.KeysplittingMessage) error {
	if err := d.keysplitting.Validate(keysplittingMessage); err != nil {
		return fmt.Errorf("invalid keysplitting message: %v", err.Error())
	}

	switch keysplittingMessage.Type {
	case ksmsg.Syn:
		synPayload := keysplittingMessage.KeysplittingPayload.(ksmsg.SynPayload)
		// Grab user's action
		if x := strings.Split(synPayload.Action, "/"); len(x) <= 1 {
			return fmt.Errorf("malformed action: %v", synPayload.Action)
		} else {
			// Start plugin
			if err := d.startPlugin(plgn.PluginName(x[0])); err != nil {
				return err
			}

			// Build reply message with empty payload
			if respKSMessage, err := d.keysplitting.BuildResponse(keysplittingMessage, "", []byte{}); err != nil {
				return fmt.Errorf("could not build response message: %s", err.Error())
			} else {
				d.Send(wsmsg.Keysplitting, respKSMessage)
			}
		}
	case ksmsg.Data:
		dataPayload := keysplittingMessage.KeysplittingPayload.(ksmsg.DataPayload)

		// Send message to plugin and catch response action payload
		if _, returnPayload, err := d.plugin.InputMessageHandler(dataPayload.Action, dataPayload.ActionPayload); err == nil {

			// Build and send response
			if respKSMessage, err := d.keysplitting.BuildResponse(keysplittingMessage, dataPayload.Action, returnPayload); err != nil {
				return fmt.Errorf("could not build response message: %s", err.Error())
			} else {
				d.Send(wsmsg.Keysplitting, respKSMessage)
			}
		} else {
			return err
		}
	default:
		return fmt.Errorf("invalid Keysplitting Payload")
	}
	return nil
}

func (d *DataChannel) startPlugin(plugin plgn.PluginName) error {
	msg := fmt.Sprintf("Starting %v plugin", plugin)
	d.logger.Info(msg)

	switch plugin {
	case plgn.Kube:

		// create channel and listener and pass it to the new plugin
		ch := make(chan smsg.StreamMessage)
		go func() {
			for {
				select {
				case <-d.ctx.Done():
					return
				case streamMessage := <-ch:
					d.Send(wsmsg.Stream, streamMessage)
				}
			}
		}()

		subLogger := d.logger.GetPluginLogger(plugin)
		d.plugin = kube.NewPlugin(d.ctx, subLogger, ch, d.role)
		d.logger.Info("Plugin started!")
		return nil
	default:
		return fmt.Errorf("tried to start an unhandled plugin")
	}
}
