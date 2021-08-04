package datachannel

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	wsmsg "bastionzero.com/bctl/v1/bzerolib/channels/message"
	ws "bastionzero.com/bctl/v1/bzerolib/channels/websocket"
	bzc "bastionzero.com/bctl/v1/bzerolib/keysplitting/bzcert"
	ksmsg "bastionzero.com/bctl/v1/bzerolib/keysplitting/message"
	plgn "bastionzero.com/bctl/v1/bzerolib/plugin"
	kube "bastionzero.com/bctl/v1/bzerolib/plugin/kube"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

type IDataChannel interface {
	SendAgentMessage(messageType wsmsg.MessageType, messagePayload interface{}) error
	//ProcessInputBuffer()
	InputMessageHandler(agentMessage wsmsg.AgentMessage) error
}

type DataChannel struct {
	websocket *ws.Websocket // We want to interact via interface, not directly with struct
	// inputBuffer list.List
	ksHandshake bool
	plugin      plgn.IPlugin

	// Keysplitting variables
	HPointer         string
	ExpectedHPointer string
	bzeCerts         map[string]bzc.BZCertMetadata

	publickey  string
	privatekey string
}

func NewDataChannel(serviceUrl string, hubEndpoint string, params map[string]string, headers map[string]string) (IDataChannel, error) {
	wsClient, err := ws.NewWebsocket(serviceUrl, hubEndpoint, params, headers)
	if err != nil {
		return &DataChannel{}, fmt.Errorf(err.Error())
	}

	ret := &DataChannel{
		websocket: wsClient,
		//inputBuffer: *list.New(),
		ksHandshake: false,
		publickey:   "legitkey",
	}

	// ONLY FOR TESTING PURPOSES
	ret.startPlugin("start/kube")

	// Subscribe to our input channel
	go func() {
		for {
			select {
			case agentMessage := <-ret.websocket.InputChannel:
				// v := ret.inputBuffer.PushBack(agentMessage)
				if err := ret.InputMessageHandler(agentMessage); err != nil {
					log.Printf(err.Error())
				}
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

	log.Printf("Return payload: %+v", messagePayload)

	// Push message to websocket channel output
	d.websocket.OutputChannel <- agentMessage
	return nil
}

// func (d *DataChannel) ProcessInputBuffer(agentMessage wsmsg.AgentMessage) {

// 	if d.inputBuffer.Len() == 0 {
// 		return
// 	}

// 	agentMessage := d.inputBuffer.Front()
// 	log.Printf("buffer length: %v, returned object of type: %T with values: %+v", d.inputBuffer.Len(), agentMessage, agentMessage)
// 	d.InputMessageHandler(agentMessage.Value.(wsmsg.AgentMessage))
// 	d.inputBuffer.Remove(agentMessage)

// 	if d.inputBuffer.Len() > 0 {
// 		d.ProcessInputBuffer()
// 	}
// }

func (d *DataChannel) InputMessageHandler(agentMessage wsmsg.AgentMessage) error {
	log.Printf("Received %v message", wsmsg.MessageType(agentMessage.MessageType))
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
		break
	default:
		// put all original logic here
		break
	}
	return nil
}

func (d *DataChannel) handleKeysplittingMessage(keysplittingMessage *ksmsg.KeysplittingMessage) error {
	log.Printf("Received %v message", keysplittingMessage.Type)

	switch keysplittingMessage.Type {
	case ksmsg.Syn:
		break
	case ksmsg.SynAck:
		break
	case ksmsg.Data:
		dataPayload := keysplittingMessage.KeysplittingPayload.(ksmsg.DataPayload)
		log.Printf("Received %v action data message: %+v", dataPayload.Action, dataPayload)

		if d.plugin != nil {
			// Get action
			if x := strings.Split(dataPayload.Action, "/"); len(x) <= 1 {
				return fmt.Errorf("Malformed action: %v", dataPayload.Action)
			} else {
				// Make sure current datachannel plugin matches plugin specified in action
				if plgn.PluginName(x[0]) == d.plugin.GetName() {

					// Send message to be handled by plugin and catch response action payload
					if returnPayload, err := d.plugin.InputMessageHandler(dataPayload.Action, dataPayload.ActionPayload); err == nil {
						// Build and send response
						if respKSMessage, err := keysplittingMessage.BuildResponse(returnPayload, d.publickey, d.privatekey); err != nil {
							return fmt.Errorf("Could not build response message: %s", err.Error())
						} else {
							d.SendAgentMessage(wsmsg.Keysplitting, respKSMessage)
						}
					} else {
						return err
					}
				} else {
					return fmt.Errorf("Action not intended for the plugin started in this channel")
				}
			}
		} else {
			// If there is no started plugin, check to see if this action starts one
			if strings.HasPrefix(dataPayload.Action, "start") {
				if err := d.startPlugin(dataPayload.Action); err != nil {
					return err
				}
				if respKSMessage, err := keysplittingMessage.BuildResponse("", d.publickey, d.privatekey); err != nil {
					return fmt.Errorf("Could not build response message: %s", err.Error())
				} else {
					d.SendAgentMessage(wsmsg.Keysplitting, respKSMessage)
				}
			} else {
				return fmt.Errorf("Must start plugin before sending messages to it")
			}
		}
	case ksmsg.DataAck:
		break
	default:
		return fmt.Errorf("Invalid Keysplitting Payload")
	}
	return nil
}

func (d *DataChannel) startPlugin(action string) error {
	if x := strings.Split(action, "/"); len(x) > 1 {
		pluginName := plgn.PluginName(x[1])

		log.Printf("Starting %v plugin", pluginName)
		switch pluginName {
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

			d.plugin = kube.NewPlugin(ch)
			log.Printf("Plugin started!")
		default:
			return fmt.Errorf("Tried to start an unhandled plugin")
		}
	} else {
		return fmt.Errorf("Malformed start plugin request")
	}
	return nil
}
