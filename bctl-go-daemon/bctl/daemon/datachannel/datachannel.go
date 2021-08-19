package datachannel

import (
	"encoding/json"
	"fmt"
	"log"

	ks "bastionzero.com/bctl/v1/bctl/daemon/keysplitting"
	kube "bastionzero.com/bctl/v1/bctl/daemon/plugin/kube"
	wsmsg "bastionzero.com/bctl/v1/bzerolib/channels/message"
	ws "bastionzero.com/bctl/v1/bzerolib/channels/websocket"
	ksmsg "bastionzero.com/bctl/v1/bzerolib/keysplitting/message"
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
	plugin       plgn.IPlugin
	keysplitting ks.IKeysplitting

	// Kube-specific vars aka to-be-removed
	role string
}

func NewDataChannel(configPath string,
	role string,
	serviceUrl string,
	hubEndpoint string,
	params map[string]string,
	headers map[string]string,
	targetSelectHandler func(msg wsmsg.AgentMessage) (string, error),
	autoReconnect bool) (*DataChannel, error) {

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
	}

	// Subscribe to our input channel
	go func() {
		for {
			select {
			case agentMessage := <-ret.websocket.InputChannel:
				// Handle each message in its own thread
				go func() {
					if err := ret.InputMessageHandler(agentMessage); err != nil {
						log.Print(err.Error())
					}
				}()
			case <-ret.websocket.DoneChannel:
				// The websocket has been closed
				log.Println("Websocket has been closed, closing datachannel")
				return
			}
		}
	}()

	return ret, nil
}

func (d *DataChannel) StartKubeDaemonPlugin(localhostToken string, daemonPort string, certPath string, keyPath string) error {
	if plugin, err := kube.NewKubeDaemonPlugin(localhostToken, daemonPort, certPath, keyPath); err != nil {
		return fmt.Errorf("could not start kube daemon plugin: %v", err.Error())
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
		log.Printf("Error building Syn: %v", err.Error())
		return
	} else {
		d.SendAgentMessage(wsmsg.Keysplitting, synMessage)
	}
}

func (d *DataChannel) InputMessageHandler(agentMessage wsmsg.AgentMessage) error {
	log.Printf("Datachannel received %v message", wsmsg.MessageType(agentMessage.MessageType))
	switch wsmsg.MessageType(agentMessage.MessageType) {
	case wsmsg.Keysplitting:
		var ksMessage ksmsg.KeysplittingMessage
		if err := json.Unmarshal(agentMessage.MessagePayload, &ksMessage); err != nil {
			return fmt.Errorf("malformed Keysplitting Message")
		} else {
			if err := d.handleKeysplittingMessage(&ksMessage); err != nil {
				return err
			}
		}
	case wsmsg.Stream:
		var sMessage smsg.StreamMessage
		log.Printf("STREAM BEFORE UNMARSHAL: %v", string(agentMessage.MessagePayload))
		if err := json.Unmarshal(agentMessage.MessagePayload, &sMessage); err != nil {
			return fmt.Errorf("malformed Stream Message")
		} else {
			if err := d.plugin.PushStreamInput(sMessage); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unhandled Message type: %v", agentMessage.MessageType)
	}
	return nil
}

// TODO: simplify this and have them both deserialize into a "common keysplitting" message
func (d *DataChannel) handleKeysplittingMessage(keysplittingMessage *ksmsg.KeysplittingMessage) error {
	if err := d.keysplitting.Validate(keysplittingMessage); err != nil {
		return fmt.Errorf("invalid keysplitting message: %v", err.Error())
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
		return fmt.Errorf("unhandled Keysplitting type")
	}

	// Send message to plugin's input message handler
	if action, returnPayload, err := d.plugin.InputMessageHandler(action, actionResponsePayload); err == nil {

		// Build and send response
		if respKSMessage, err := d.keysplitting.BuildResponse(keysplittingMessage, action, returnPayload); err != nil {
			return fmt.Errorf("could not build response message: %s", err.Error())
		} else {
			d.SendAgentMessage(wsmsg.Keysplitting, respKSMessage)
			return nil
		}
	} else {
		return err
	}

	// switch keysplittingMessage.Type {
	// case ksmsg.SynAck:
	// 	synAckPayload := keysplittingMessage.KeysplittingPayload.(ksmsg.SynAckPayload)
	// 	if action, returnPayload, err := d.plugin.InputMessageHandler(synAckPayload.Action, synAckPayload.ActionResponsePayload); err == nil {

	// 		// Build and send response
	// 		if respKSMessage, err := d.keysplitting.BuildResponse(keysplittingMessage, action, returnPayload); err != nil {
	// 			return fmt.Errorf("could not build response message: %s", err.Error())
	// 		} else {
	// 			d.SendAgentMessage(wsmsg.Keysplitting, respKSMessage)
	// 		}
	// 	} else {
	// 		return err
	// 	}
	// case ksmsg.DataAck:
	// 	dataAckPayload := keysplittingMessage.KeysplittingPayload.(ksmsg.DataAckPayload)
	// 	if action, returnPayload, err := d.plugin.InputMessageHandler(dataAckPayload.Action, dataAckPayload.ActionResponsePayload); err == nil {

	// 		// Build and send response
	// 		if respKSMessage, err := d.keysplitting.BuildResponse(keysplittingMessage, action, returnPayload); err != nil {
	// 			return fmt.Errorf("could not build response message: %s", err.Error())
	// 		} else {
	// 			d.SendAgentMessage(wsmsg.Keysplitting, respKSMessage)
	// 		}
	// 	} else {
	// 		return err
	// 	}
	// default:
	// 	return fmt.Errorf("invalid keysplitting payload")
	// }
	// return nil
}
