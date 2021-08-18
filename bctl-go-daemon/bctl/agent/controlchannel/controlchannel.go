package controlchannel

import (
	ed "crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"bastionzero.com/bctl/v1/bctl/agent/vault"
	wsmsg "bastionzero.com/bctl/v1/bzerolib/channels/message"
	ws "bastionzero.com/bctl/v1/bzerolib/channels/websocket"
)

const (
	hubEndpoint   = "/api/v1/hub/kube-control"
	autoReconnect = true
)

type ControlChannel struct {
	websocket *ws.Websocket

	// These are all the types of channels we have available
	NewDatachannelChan chan NewDatachannelMessage

	SocketLock sync.Mutex // Ref: https://github.com/gorilla/websocket/issues/119#issuecomment-198710015
}

// Constructor to create a new Control Websocket Client
func NewControlChannel(serviceUrl string,
	activationToken string,
	orgId string,
	clusterName string,
	environmentId string,
	agentVersion string,
	targetSelectHandler func(msg wsmsg.AgentMessage) (string, error)) (*ControlChannel, error) {

	// Create our headers and params, headers are empty
	headers := make(map[string]string)

	// Make and add our params
	params := make(map[string]string)
	params["activation_token"] = activationToken
	params["org_id"] = orgId
	params["cluster_name"] = clusterName
	params["environment_id"] = environmentId
	params["agent_version"] = agentVersion // TODO: this should come from somewhere else...

	log.Printf("\nserviceURL: %v, \nhubEndpoint: %v, \nparams: %v, \nheaders: %v", serviceUrl, hubEndpoint, params, headers)

	wsClient, err := ws.NewWebsocket(serviceUrl, hubEndpoint, params, headers, targetSelectHandler, autoReconnect)
	if err != nil {
		return &ControlChannel{}, err
	}

	// populate keys if they haven't been generated already
	// TODO: revisit this
	if err := newAgent(); err != nil {
		return &ControlChannel{}, err
	}

	control := ControlChannel{
		websocket:          wsClient,
		NewDatachannelChan: make(chan NewDatachannelMessage),
	}

	// Set up our handler to deal with incoming messages
	go func() {
		for {
			select {
			case <-control.websocket.DoneChannel:
				log.Println("Websocket has been closed, closing datachannel")
				return
			case agentMessage := <-control.websocket.InputChannel:
				switch wsmsg.MessageType(agentMessage.MessageType) {
				case wsmsg.NewDatachannel:
					var dataMessage NewDatachannelMessage
					if err := json.Unmarshal(agentMessage.MessagePayload, &dataMessage); err != nil {
						log.Printf("Could not unmarshal new datachannel request: %v", err.Error())
						return
					} else {
						control.NewDatachannelChan <- dataMessage
					}
				case wsmsg.HealthCheck:
					control.websocket.OutputChannel <- wsmsg.AgentMessage{
						MessageType:    string(wsmsg.HealthCheck),
						SchemaVersion:  wsmsg.SchemaVersion,
						MessagePayload: []byte{}, // Double check we send an empty message
					}
				}
			}
		}
	}()

	return &control, nil
}

func newAgent() error {
	config, _ := vault.LoadVault()

	// Check if vault is empty, if not generate a private, public key pair
	if config.IsEmpty() {
		if publicKey, privateKey, err := ed.GenerateKey(nil); err != nil {
			return fmt.Errorf("error generating key pair: %v", err.Error())
		} else {
			pubkeyString := base64.StdEncoding.EncodeToString([]byte(publicKey))
			privkeyString := base64.StdEncoding.EncodeToString([]byte(privateKey))
			config.Data = vault.SecretData{
				PublicKey:  pubkeyString,
				PrivateKey: privkeyString,
			}
			if err := config.Save(); err != nil {
				return fmt.Errorf("error saving vault: %v", err.Error())
			} else {
				return nil
			}
		}
	} else {
		// If the vault isn't empty, don't do anything
		log.Printf("config data: %+v", config.Data)
		return nil
	}
}
