package datachannel

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	kube "bastionzero.com/bctl/v1/bctl/agent/plugin/kube"
	wsmsg "bastionzero.com/bctl/v1/bzerolib/channels/message"
	ws "bastionzero.com/bctl/v1/bzerolib/channels/websocket"
	ks "bastionzero.com/bctl/v1/bzerolib/keysplitting"
	ksmsg "bastionzero.com/bctl/v1/bzerolib/keysplitting/message"
	plgn "bastionzero.com/bctl/v1/bzerolib/plugin"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

type IDataChannel interface {
	SendAgentMessage(messageType wsmsg.MessageType, messagePayload interface{}) error
	InputMessageHandler(agentMessage wsmsg.AgentMessage) error
	// handlekeysplittingmessage for when we break things out
	StartKubeDaemonPlugin(localhostToken string, daemonPort string, certPath string, keyPath string) error
}

type DataChannel struct {
	websocket    *ws.Websocket
	plugin       plgn.IPlugin
	keysplitting ks.IKeysplitting

	// Kube-specific vars aka to-be-removed
	role string
}

func NewDataChannel(role string, startPlugin string, serviceUrl string, hubEndpoint string, params map[string]string, headers map[string]string, targetSelectHandler func(msg wsmsg.AgentMessage) (string, error), autoReconnect bool) (*DataChannel, error) {
	wsClient, err := ws.NewWebsocket(serviceUrl, hubEndpoint, params, headers, targetSelectHandler, autoReconnect)
	if err != nil {
		return &DataChannel{}, err // TODO: how tf are we going to report these?
	}

	keysplitter, err := ks.NewKeysplitting("", configPath)
	if err != nil {
		return &DataChannel{}, err
	}
	ret := &DataChannel{
		websocket:    wsClient,
		keysplitting: keysplitter,
		role:         role,
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

func (d *DataChannel) SendSyn() { // TODO: have this return an error
	payload := map[string]string{
		"Role": d.role,
	}
	payloadBytes, _ := json.Marshal(payload)

	if synMessage, err := d.keysplitting.BuildSyn("kube/restapi", payloadBytes); err != nil {
		log.Printf("Error building Syn: %v", err.Error())
		return
	} else {
		d.SendAgentMessage(wsmsg.Keysplitting, synMessage)
	}
	// // useful for when we implement keysplitting, for now just helps with daemon startup
	// if action, payload, err := d.plugin.InputMessageHandler("", []byte{}); err != nil {
	// 	log.Printf(err.Error())
	// } else {
	// 	log.Printf("YEAHHHHHH BITCH")
	// 	// log.Printf("Action: %v, Payload: %v", action, payload)

	// 	// should only be building this message once and it should only be the syn and it should be in a helper function
	// 	// this is a temporary workaround for getting the daemon working
	// 	dataPayload := ksmsg.DataPayload{
	// 		Timestamp:     "",
	// 		SchemaVersion: "zero",
	// 		Type:          "Data",
	// 		Action:        action,
	// 		TargetId:      "",
	// 		HPointer:      "",
	// 		BZCertHash:    "",
	// 		ActionPayload: payload,
	// 	}
	// 	ksMessage := ksmsg.KeysplittingMessage{
	// 		Type:                "Data",
	// 		KeysplittingPayload: dataPayload,
	// 		Signature:           "",
	// 	}
	// 	d.SendAgentMessage(wsmsg.Keysplitting, ksMessage)
	// }
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
	if err := d.keysplitting.Validate(keysplittingMessage); err != nil {
		return fmt.Errorf("Invalid keysplitting message: %v", err.Error())
	}

	switch keysplittingMessage.Type {
	case ksmsg.Syn:
		synPayload := keysplittingMessage.KeysplittingPayload.(ksmsg.SynPayload)
		// Start the plugin that will execute the action
		if x := strings.Split(synPayload.Action, "/"); len(x) <= 1 {
			return fmt.Errorf("Malformed action: %v", synPayload.Action)
		} else {
			// Start plugin
			if err := d.startPlugin(plgn.PluginName(x[0])); err != nil {
				return err
			}

			// Build reply message with empty payload
			if respKSMessage, err := d.keysplitting.BuildResponse(keysplittingMessage, "", []byte{}); err != nil {
				return fmt.Errorf("Could not build response message: %s", err.Error())
			} else {
				d.SendAgentMessage(wsmsg.Keysplitting, respKSMessage)
			}
		}
	case ksmsg.SynAck:
		synAckPayload := keysplittingMessage.KeysplittingPayload.(ksmsg.SynAckPayload)
		if action, returnPayload, err := d.plugin.InputMessageHandler(synAckPayload.Action, synAckPayload.ActionResponsePayload); err == nil {

			// Build and send response
			if respKSMessage, err := d.keysplitting.BuildResponse(keysplittingMessage, action, returnPayload); err != nil {
				return fmt.Errorf("Could not build response message: %s", err.Error())
			} else {
				d.SendAgentMessage(wsmsg.Keysplitting, respKSMessage)
			}
		} else {
			return err
		}
	case ksmsg.Data:
		dataPayload := keysplittingMessage.KeysplittingPayload.(ksmsg.DataPayload)

		// Send message to plugin and catch response action payload
		if _, returnPayload, err := d.plugin.InputMessageHandler(dataPayload.Action, dataPayload.ActionPayload); err == nil {

			// Build and send response
			if respKSMessage, err := d.keysplitting.BuildResponse(keysplittingMessage, dataPayload.Action, returnPayload); err != nil {
				return fmt.Errorf("Could not build response message: %s", err.Error())
			} else {
				d.SendAgentMessage(wsmsg.Keysplitting, respKSMessage)
			}
		} else {
			return err
		}
	case ksmsg.DataAck:
		dataAckPayload := keysplittingMessage.KeysplittingPayload.(ksmsg.DataAckPayload)
		if action, returnPayload, err := d.plugin.InputMessageHandler(dataAckPayload.Action, dataAckPayload.ActionResponsePayload); err == nil {

			// Build and send response
			if respKSMessage, err := d.keysplitting.BuildResponse(keysplittingMessage, action, returnPayload); err != nil {
				return fmt.Errorf("Could not build response message: %s", err.Error())
			} else {
				d.SendAgentMessage(wsmsg.Keysplitting, respKSMessage)
			}
		} else {
			return err
		}
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
