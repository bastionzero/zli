package datachannel

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	kube "bastionzero.com/bctl/v1/bctl/agent/plugin/kube"
	wsmsg "bastionzero.com/bctl/v1/bzerolib/channels/message"
	ws "bastionzero.com/bctl/v1/bzerolib/channels/websocket"
	bzc "bastionzero.com/bctl/v1/bzerolib/keysplitting/bzcert"
	ksmsg "bastionzero.com/bctl/v1/bzerolib/keysplitting/message"
	plgn "bastionzero.com/bctl/v1/bzerolib/plugin"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

type IDataChannel interface {
	SendAgentMessage(messageType wsmsg.MessageType, messagePayload interface{}) error
	InputMessageHandler(agentMessage wsmsg.AgentMessage) error
}

type DataChannel struct {
	websocket *ws.Websocket // We want to interact via interface, not directly with struct
	// inputBuffer list.List
	ksHandshake bool
	plugin      plgn.IPlugin

	// Kube-specific vars
	role string

	// Keysplitting variables
	HPointer         string
	ExpectedHPointer string
	bzeCerts         map[string]bzc.BZCertMetadata
	publickey        string
	privatekey       string
}

func NewDataChannel(role string, startPlugin string, serviceUrl string, hubEndpoint string, params map[string]string, headers map[string]string, targetSelectHandler func(msg wsmsg.AgentMessage) (string, error), autoReconnect bool) (*DataChannel, error) {
	wsClient, err := ws.NewWebsocket(serviceUrl, hubEndpoint, params, headers, targetSelectHandler, autoReconnect)
	if err != nil {
		return &DataChannel{}, fmt.Errorf(err.Error())
	}

	ret := &DataChannel{
		websocket:   wsClient,
		ksHandshake: false,
		publickey:   "legitkey",
		privatekey:  "equallylegitkey",
		role:        role,
	}

	// Start plugin on startup, if specified
	ret.startPlugin(plgn.PluginName(startPlugin))

	// Subscribe to our input channel
	go func() {
		for {
			select {
			case agentMessage := <-ret.websocket.InputChannel:
				// Handle each message in its own thread
				go func() {
					if err := ret.InputMessageHandler(agentMessage); err != nil {
						log.Printf(err.Error())
					}
				}()
				break
			case <-ret.websocket.DoneChannel:
				// The websocket has been closed
				log.Println("Websocket has been closed, closing datachannel")
				return
			}
		}
	}()

	return ret, nil
}

// Wraps and sends the payload
func (d *DataChannel) SendAgentMessage(messageType wsmsg.MessageType, messagePayload interface{}) error {
	messageBytes, _ := json.Marshal(messagePayload)
	agentMessage := wsmsg.AgentMessage{
		MessageType:    string(messageType),
		SchemaVersion:  wsmsg.Schema,
		MessagePayload: messageBytes,
	}

	// Push message to websocket channel output
	d.websocket.OutputChannel <- agentMessage
	return nil
}

func (d *DataChannel) SendSyn() {
	// useful for when we implement keysplitting, for now just helps with daemon startup
	if action, payload, err := d.plugin.InputMessageHandler("", []byte{}); err != nil {
		log.Printf(err.Error())
	} else {
		log.Printf("Action: %v", action)

		// should only be building this message once and it should only be the syn and it should be in a helper function
		// this is a temporary workaround for getting the daemon working
		dataPayload := ksmsg.DataPayload{
			Timestamp:     "",
			SchemaVersion: "zero",
			Type:          "Data",
			Action:        action,
			TargetId:      "",
			HPointer:      "",
			BZCertHash:    "",
			ActionPayload: payload,
		}
		ksMessage := ksmsg.KeysplittingMessage{
			Type:                "Data",
			KeysplittingPayload: dataPayload,
			Signature:           "",
		}
		d.SendAgentMessage(wsmsg.Keysplitting, ksMessage)
	}
}

func (d *DataChannel) InputMessageHandler(agentMessage wsmsg.AgentMessage) error {
	log.Printf("Datachannel received %v message", wsmsg.MessageType(agentMessage.MessageType))
	switch wsmsg.MessageType(agentMessage.MessageType) {
	case wsmsg.Keysplitting:
		var ksMessage ksmsg.KeysplittingMessage
		if err := json.Unmarshal(agentMessage.MessagePayload, &ksMessage); err != nil {
			return fmt.Errorf("Malformed Keysplitting Message")
		} else {
			if err := d.handleKeysplittingMessage(&ksMessage); err != nil {
				return err
			}
		}
	case wsmsg.Stream:
		var sMessage smsg.StreamMessage
		if err := json.Unmarshal(agentMessage.MessagePayload, &sMessage); err != nil {
			return fmt.Errorf("Malformed Stream Message")
		} else {
			if err := d.plugin.PushStreamInput(sMessage); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("Unhandled Message type: %v", agentMessage.MessageType)
	}
	return nil
}

func (d *DataChannel) handleKeysplittingMessage(keysplittingMessage *ksmsg.KeysplittingMessage) error {
	switch keysplittingMessage.Type {
	case ksmsg.Syn:
		break
	case ksmsg.SynAck:
		// in the future we call inputmessagehandler for the daemon from here
		break
	case ksmsg.Data:
		dataPayload := keysplittingMessage.KeysplittingPayload.(ksmsg.DataPayload)

		// Figure out what action we're taking
		if x := strings.Split(dataPayload.Action, "/"); len(x) <= 1 {
			return fmt.Errorf("Malformed action: %v", dataPayload.Action)
		} else {
			if d.plugin != nil {
				// Send message to be handled by plugin and catch response action payload
				if _, returnPayload, err := d.plugin.InputMessageHandler(dataPayload.Action, dataPayload.ActionPayload); err == nil {

					// Build and send response
					if respKSMessage, err := keysplittingMessage.BuildResponse("", returnPayload, d.publickey, d.privatekey); err != nil {
						return fmt.Errorf("Could not build response message: %s", err.Error())
					} else {
						d.SendAgentMessage(wsmsg.Keysplitting, respKSMessage)
					}
				} else {
					return err
				}
			} else {

				// If there is no started plugin, check to see if this action starts one
				if x[0] == "start" {

					// Start plugin
					if err := d.startPlugin(plgn.PluginName(x[1])); err != nil {
						return err
					}

					// Build reply message with empty payload
					if respKSMessage, err := keysplittingMessage.BuildResponse("", []byte{}, d.publickey, d.privatekey); err != nil {
						return fmt.Errorf("Could not build response message: %s", err.Error())
					} else {
						d.SendAgentMessage(wsmsg.Keysplitting, respKSMessage)
					}
				} else {
					return fmt.Errorf("Must start a plugin before sending messages to it")
				}
			}
		}
	case ksmsg.DataAck:
		dataAckPayload := keysplittingMessage.KeysplittingPayload.(ksmsg.DataAckPayload)
		if action, returnPayload, err := d.plugin.InputMessageHandler(dataAckPayload.Action, dataAckPayload.ActionResponsePayload); err == nil {

			// Build and send response
			if respKSMessage, err := keysplittingMessage.BuildResponse(action, returnPayload, d.publickey, d.privatekey); err != nil {
				return fmt.Errorf("Could not build response message: %s", err.Error())
			} else {
				d.SendAgentMessage(wsmsg.Keysplitting, respKSMessage)
			}
		} else {
			return err
		}
		break
	default:
		return fmt.Errorf("Invalid Keysplitting Payload")
	}
	return nil
}

func (d *DataChannel) startPlugin(plugin plgn.PluginName) error {
	log.Printf("Starting %v plugin", plugin)
	switch plugin {
	case plgn.Kube:

		// create channel and listener and pass it to the new plugin
		ch := make(chan smsg.StreamMessage)
		go func() {
			for {
				select {
				case streamMessage := <-ch:
					d.SendAgentMessage(wsmsg.Stream, streamMessage)
				}
			}
		}()

		d.plugin = kube.NewPlugin(ch, d.role)
		log.Printf("Plugin started!")
		return nil
	default:
		return fmt.Errorf("Tried to start an unhandled plugin")
	}
}
