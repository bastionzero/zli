package controlchannel

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sync"

	"bastionzero.com/bctl/v1/bctl/agent/vault"
	wsmsg "bastionzero.com/bctl/v1/bzerolib/channels/message"
	ws "bastionzero.com/bctl/v1/bzerolib/channels/websocket"

	ed "crypto/ed25519"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	hubEndpoint      = "/api/v1/hub/kube-control"
	registerEndpoint = "/api/v1/kube/register-agent"
	autoReconnect    = true
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

	// Populate keys if they haven't been generated already
	config, err := newAgent(serviceUrl, activationToken, agentVersion, orgId, environmentId, clusterName)
	if err != nil {
		return &ControlChannel{}, err
	}

	// Create our headers and params, headers are empty
	headers := make(map[string]string)

	// Make and add our params
	params := make(map[string]string)
	params["public_key"] = config.Data.PublicKey
	params["org_id"] = orgId
	params["cluster_name"] = clusterName
	params["environment_id"] = environmentId
	params["agent_version"] = agentVersion

	log.Printf("\nserviceURL: %v, \nhubEndpoint: %v, \nparams: %v, \nheaders: %v", serviceUrl, hubEndpoint, params, headers)

	wsClient, err := ws.NewWebsocket(serviceUrl, hubEndpoint, params, headers, targetSelectHandler, autoReconnect, true)
	if err != nil {
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
					if msg, err := aliveCheck(); err != nil {
						log.Printf(err.Error())
						return
					} else {
						control.websocket.OutputChannel <- wsmsg.AgentMessage{
							MessageType:    string(wsmsg.HealthCheck),
							SchemaVersion:  wsmsg.SchemaVersion,
							MessagePayload: msg,
						}
					}
				}
			}
		}
	}()
	return &control, nil
}

func aliveCheck() ([]byte, error) {
	// Also let bastion know a list of valid cluster roles
	// Create our api object
	config, err := rest.InClusterConfig()
	if err != nil {
		return []byte{}, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return []byte{}, err
	}

	// Then get all cluster roles
	clusterRoleBindings, err := clientset.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return []byte{}, err
	}

	clusterUsers := make(map[string]bool)

	for _, clusterRoleBinding := range clusterRoleBindings.Items {
		// Now loop over the subjects if we can find any user subjects
		for _, subject := range clusterRoleBinding.Subjects {
			if subject.Kind == "User" {
				// We do not consider any system:... or eks:..., basically any system: looking roles as valid. This can be overridden from Bastion
				var systemRegexPatten = regexp.MustCompile(`[a-zA-Z0-9]*:[a-za-zA-Z0-9-]*`)
				if !systemRegexPatten.MatchString(subject.Name) {
					clusterUsers[subject.Name] = true
				}
			}
		}
	}

	// Then get all roles
	roleBindings, err := clientset.RbacV1().RoleBindings("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return []byte{}, err
	}

	for _, roleBindings := range roleBindings.Items {
		// Now loop over the subjects if we can find any user subjects
		for _, subject := range roleBindings.Subjects {
			if subject.Kind == "User" {
				// We do not consider any system:... or eks:..., basically any system: looking roles as valid. This can be overridden from Bastion
				var systemRegexPatten = regexp.MustCompile(`[a-zA-Z0-9]*:[a-za-zA-Z0-9-]*`) // TODO: double check
				if !systemRegexPatten.MatchString(subject.Name) {
					clusterUsers[subject.Name] = true
				}
			}
		}
	}

	// Now build our response
	users := []string{}
	for key := range clusterUsers {
		users = append(users, key)
	}

	alive := AliveCheckClusterToBastionMessage{
		Alive:        true,
		ClusterUsers: users,
	}

	aliveBytes, _ := json.Marshal(alive)
	return aliveBytes, nil
}

func newAgent(serviceUrl string, activationToken string, agentVersion string, orgId string, environmentId string, clusterName string) (*vault.Vault, error) {
	config, _ := vault.LoadVault()

	// Check if vault is empty, if so generate a private, public key pair
	if config.IsEmpty() {
		log.Println("Creating new agent secret")

		if publicKey, privateKey, err := ed.GenerateKey(nil); err != nil {
			return nil, fmt.Errorf("error generating key pair: %v", err.Error())
		} else {
			pubkeyString := base64.StdEncoding.EncodeToString([]byte(publicKey))
			privkeyString := base64.StdEncoding.EncodeToString([]byte(privateKey))
			config.Data = vault.SecretData{
				PublicKey:  pubkeyString,
				PrivateKey: privkeyString,
			}
			if err := config.Save(); err != nil {
				return nil, fmt.Errorf("error saving vault: %v", err.Error())
			}

			// Register with Bastion
			log.Println("Registering agent with Bastionn")
			register := RegisterAgentMessage{
				PublicKey:      pubkeyString,
				ActivationCode: activationToken,
				AgentVersion:   agentVersion,
				OrgId:          orgId,
				EnvironmentId:  environmentId,
				ClusterName:    clusterName,
			}

			registerJson, err := json.Marshal(register)
			if err != nil {
				log.Printf("Error marshalling register data")
				return nil, err
			}

			// Make our POST request
			response, err := http.Post("https://"+serviceUrl+registerEndpoint, "application/json",
				bytes.NewBuffer(registerJson))
			if err != nil || response.StatusCode != http.StatusOK {
				log.Printf("Error making post request to register agent. Error: %s. Response: %s", err, response)
				return nil, err
			}
		}
	} else {
		// If the vault isn't empty, don't do anything
		log.Printf("Found Previous config data: %+v", config.Data)
	}
	return config, nil
}
