package datachannel

import (
	"encoding/json"
	"fmt"

	ks "bastionzero.com/bctl/v1/bctl/daemon/keysplitting"
	kube "bastionzero.com/bctl/v1/bctl/daemon/plugin/kube"
	wsmsg "bastionzero.com/bctl/v1/bzerolib/channels/message"
	ws "bastionzero.com/bctl/v1/bzerolib/channels/websocket"
	ksmsg "bastionzero.com/bctl/v1/bzerolib/keysplitting/message"
	lggr "bastionzero.com/bctl/v1/bzerolib/logger"
	plgn "bastionzero.com/bctl/v1/bzerolib/plugin"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

type IDataChannel interface {
	SendAgentMessage(messageType wsmsg.MessageType, messagePayload interface{}) error
	InputMessageHandler(agentMessage wsmsg.AgentMessage) error
	StartKubeDaemonPlugin(localhostToken string, daemonPort string, certPath string, keyPath string) error
}

type DataChannel struct {
	websocket    *ws.Websocket
	logger       *lggr.Logger
	plugin       plgn.IPlugin
	keysplitting ks.IKeysplitting

	// Kube-specific vars aka to-be-removed
	role string

	// Done channel to bubble up messages to kubectl
	doneChannel chan string
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

	subLogger := logger.GetWebsocketLogger()
	wsClient, err := ws.NewWebsocket(subLogger, serviceUrl, hubEndpoint, params, headers, targetSelectHandler, autoReconnect, false)
	if err != nil {
		return &DataChannel{}, err // TODO: how tf are we going to report these?
	}

	keysplitter, err := ks.NewKeysplitting("", configPath)
	if err != nil {
		return &DataChannel{}, err
	}
	ret := &DataChannel{
		websocket:    wsClient,
		logger:       logger,
		keysplitting: keysplitter,
		doneChannel:  make(chan string),
	}

	// Subscribe to our input channel
	go func() {
		for {
			select {
			case agentMessage := <-ret.websocket.InputChannel:
				// Handle each message in its own thread
				go func() {
					if err := ret.InputMessageHandler(agentMessage); err != nil {
						ret.logger.Error(err)
					}
				}()
			case <-ret.websocket.DoneChannel:
				// The websocket has been closed
				ret.logger.Info("Websocket has been closed, closing datachannel")

				// Send a message to our done channel to the kubectl can get the message
				ret.doneChannel <- "true"
				return
			}
		}
	}()

	return ret, nil
}

func (d *DataChannel) StartKubeDaemonPlugin(localhostToken string, daemonPort string, certPath string, keyPath string) error {
	subLogger := d.logger.GetPluginLogger(plgn.KubeDaemon)
	if plugin, err := kube.NewKubeDaemonPlugin(subLogger, localhostToken, daemonPort, certPath, keyPath, d.doneChannel); err != nil {
		rerr := fmt.Errorf("could not start kube daemon plugin: %s", err)
		d.logger.Error(rerr)
		return rerr
	} else {
		d.plugin = plugin
		return nil
	}
}

// Wraps and sends the payload
func (d *DataChannel) SendAgentMessage(messageType wsmsg.MessageType, messagePayload interface{}) error {
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

func (d *DataChannel) SendSyn() { // TODO: have this return an error
	payload := map[string]string{
		"Role": d.role,
	}
	payloadBytes, _ := json.Marshal(payload)

	// TODO: have the action be something more meaningful, probably passed in as a flag
	action := "kube/restapi"
	if synMessage, err := d.keysplitting.BuildSyn(action, payloadBytes); err != nil {
		rerr := fmt.Errorf("error building Syn: %s", err)
		d.logger.Error(rerr)
		return
	} else {
		d.SendAgentMessage(wsmsg.Keysplitting, synMessage)
	}
}

func (d *DataChannel) InputMessageHandler(agentMessage wsmsg.AgentMessage) error {
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
	case ksmsg.DataAck:
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

		// Build and send response
		if respKSMessage, err := d.keysplitting.BuildResponse(keysplittingMessage, action, returnPayload); err != nil {
			rerr := fmt.Errorf("could not build response message: %s", err)
			d.logger.Error(rerr)
			return rerr
		} else {
			d.SendAgentMessage(wsmsg.Keysplitting, respKSMessage)
			return nil
		}
	} else {
		d.logger.Error(err)
		return err
	}
}
