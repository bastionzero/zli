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
	rrr "bastionzero.com/bctl/v1/bzerolib/error"
	ksmsg "bastionzero.com/bctl/v1/bzerolib/keysplitting/message"
	lggr "bastionzero.com/bctl/v1/bzerolib/logger"
	plgn "bastionzero.com/bctl/v1/bzerolib/plugin"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

type IDataChannel interface {
	Send(messageType wsmsg.MessageType, messagePayload interface{})
	Receive(agentMessage wsmsg.AgentMessage)
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

	ctx, cancel := context.WithCancel(context.Background())

	wsClient, err := ws.NewWebsocket(ctx, subLogger, serviceUrl, hubEndpoint, params, headers, targetSelectHandler, autoReconnect, false)
	if err != nil {
		cancel()
		logger.Error(err)
		return &DataChannel{}, err // TODO: how are we going to report these? control channel, bro
	}

	keysplitter, err := ks.NewKeysplitting()
	if err != nil {
		cancel()
		logger.Error(err)
		return &DataChannel{}, err
	}

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
			case agentMessage := <-ret.websocket.InputChan:
				// Handle each message in its own thread
				go func() {
					ret.Receive(agentMessage)
				}()
			case <-ret.websocket.DoneChan:
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
func (d *DataChannel) Send(messageType wsmsg.MessageType, messagePayload interface{}) {
	// Stop any further messages from being sent once context is cancelled
	if d.ctx.Err() == context.Canceled {
		return
	}

	messageBytes, _ := json.Marshal(messagePayload)
	agentMessage := wsmsg.AgentMessage{
		MessageType:    string(messageType),
		SchemaVersion:  wsmsg.SchemaVersion,
		MessagePayload: messageBytes,
	}

	// Push message to websocket channel output
	d.websocket.OutputChan <- agentMessage
}

func (d *DataChannel) sendError(errType rrr.ErrorType, err error) {
	d.logger.Error(err)
	errMsg := rrr.ErrorMessage{
		Type:     string(errType),
		Message:  err.Error(),
		HPointer: d.keysplitting.GetHpointer(),
	}
	d.Send(wsmsg.Error, errMsg)
}

func (d *DataChannel) Receive(agentMessage wsmsg.AgentMessage) {
	d.logger.Info("received message type: " + agentMessage.MessageType)

	switch wsmsg.MessageType(agentMessage.MessageType) {
	case wsmsg.Keysplitting:
		var ksMessage ksmsg.KeysplittingMessage
		if err := json.Unmarshal(agentMessage.MessagePayload, &ksMessage); err != nil {
			rerr := fmt.Errorf("malformed Keysplitting message")
			d.sendError(rrr.KeysplittingValidationError, rerr)
		} else {
			d.handleKeysplittingMessage(&ksMessage)
		}
	default:
		rerr := fmt.Errorf("unhandled message type: %v", agentMessage.MessageType)
		d.sendError(rrr.ComponentProcessingError, rerr)
	}
}

func (d *DataChannel) handleKeysplittingMessage(keysplittingMessage *ksmsg.KeysplittingMessage) {
	if err := d.keysplitting.Validate(keysplittingMessage); err != nil {
		rerr := fmt.Errorf("invalid keysplitting message: %s", err)
		d.sendError(rrr.KeysplittingValidationError, rerr)
		return
	}

	switch keysplittingMessage.Type {
	case ksmsg.Syn:
		synPayload := keysplittingMessage.KeysplittingPayload.(ksmsg.SynPayload)
		// Grab user's action
		if x := strings.Split(synPayload.Action, "/"); len(x) <= 1 {
			rerr := fmt.Errorf("malformed action: %s", synPayload.Action)
			d.sendError(rrr.KeysplittingValidationError, rerr)
			return
		} else {
			if d.plugin != nil { // Don't start plugin if there's already one started
				return
			}

			// Start plugin
			if err := d.startPlugin(plgn.PluginName(x[0])); err != nil {
				d.sendError(rrr.ComponentStartupError, err)
				return
			}

			d.sendKeysplittingMessage(keysplittingMessage, "", []byte{}) // empty payload
		}
	case ksmsg.Data:
		dataPayload := keysplittingMessage.KeysplittingPayload.(ksmsg.DataPayload)

		// Send message to plugin and catch response action payload
		if _, returnPayload, err := d.plugin.InputMessageHandler(dataPayload.Action, dataPayload.ActionPayload); err == nil {

			// Build and send response
			d.sendKeysplittingMessage(keysplittingMessage, dataPayload.Action, returnPayload)
		} else {
			rerr := fmt.Errorf("unrecognized keysplitting message type: %s", keysplittingMessage.Type)
			d.sendError(rrr.KeysplittingValidationError, rerr)
		}
	default:
		rerr := fmt.Errorf("invalid Keysplitting Payload")
		d.sendError(rrr.KeysplittingValidationError, rerr)
	}
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
