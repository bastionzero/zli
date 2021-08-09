package controlWebsocketTypes

import (
	"encoding/json"
	"log"
	"sync"
	"os"
	"bytes"
	"encoding/gob"
	"context"

	"bastionzero.com/bctl/v1/commonWebsocketClient"

	"github.com/gorilla/websocket"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreV1Types "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ProvisionNewWebsocketSignalRMessage struct {
	Target    string                         `json:"target"`
	Arguments []ProvisionNewWebsocketMessage `json:"arguments"`
	Type      int                            `json:"type"`
}

type ProvisionNewWebsocketMessage struct {
	ConnectionId string `json:"connectionId"`
	Role         string `json:"role"`
}

type AliveCheckToClusterFromBastionSignalRMessage struct {
	Target    string                                  `json:"target"`
	Arguments []AliveCheckToClusterFromBastionMessage `json:"arguments"`
	Type      int                                     `json:"type"`
}
type AliveCheckToClusterFromBastionMessage struct {
}

type AliveCheckToBastionFromClusterSignalRMessage struct {
	Target    string                                  `json:"target"`
	Arguments []AliveCheckToBastionFromClusterMessage `json:"arguments"`
	Type      int                                     `json:"type"`
}
type AliveCheckToBastionFromClusterMessage struct {
	Alive        bool     `json:"alive"`
	ClusterUsers []string `json:"clusterUsers`
}

type ControlWebsocket struct {
	WebsocketClient *commonWebsocketClient.WebsocketClient

	// These are all the    types of channels we have available
	ProvisionWebsocketChan chan ProvisionNewWebsocketMessage
	AliveCheckChan         chan AliveCheckToClusterFromBastionSignalRMessage

	SocketLock sync.Mutex // Ref: https://github.com/gorilla/websocket/issues/119#issuecomment-198710015
}

type SecretConfig struct {
	Test string
}

func (client *ControlWebsocket) SendAliveCheckToBastionFromClusterMessage(aliveCheckToBastionFromClusterMessage AliveCheckToBastionFromClusterMessage) error {
	// Lock our mutex and setup the unlock
	client.SocketLock.Lock()
	defer client.SocketLock.Unlock()

	log.Printf("Sending AliveCheck to Bastion")
	// Create the object, add relevent information
	toSend := new(AliveCheckToBastionFromClusterSignalRMessage)
	toSend.Target = "AliveCheckToBastionFromCluster"
	toSend.Arguments = []AliveCheckToBastionFromClusterMessage{aliveCheckToBastionFromClusterMessage}

	// Add the type number from the class
	toSend.Type = 1 // Ref: https://github.com/aspnet/SignalR/blob/master/specs/HubProtocol.md#invocation-message-encoding

	// Marshal our message
	toSendMarshalled, err := json.Marshal(toSend)
	if err != nil {
		return err
	}

	// Write our message
	if err = client.WebsocketClient.Client.WriteMessage(websocket.TextMessage, append(toSendMarshalled, 0x1E)); err != nil {
		return err
	}
	// client.SignalRTypeNumber++
	return nil
}

func (client *ControlWebsocket) NewAgent() bool {
	// Helper function to determin if this is a fresh install of an agent
	// This works by checking the secret value 
	secret := client.GetSecret()

	// Now check the value of the secret
	if (bytes.Compare(secret.Data["secret"], []byte("coolbeans")) == 0) {
		return true
	}
	return false
}

func (client *ControlWebsocket) GetParsedSecret() SecretConfig {
	// First get the kube secret
	secret := client.GetSecret()
	
	// Now parse the bytes and get the secret data
	secretConfig := client.DecodeToSecretConfig(secret.Data["secret"])

	return secretConfig
}

func (client *ControlWebsocket) SaveParsedSecret(secretConfig SecretConfig) {
	// First get the kube secret
	secret := client.GetSecret()

	// Now encode the secretConfig
	encodedSecretConfig := client.EncodeToBytes(secretConfig)

	// Now update the kube secret object
	secret.Data["secret"] = encodedSecretConfig

	// Now update the secret
	client.UpdateSecret(secret)
}

func (client *ControlWebsocket) GetSecret() *coreV1.Secret {
	// Build our client
	secretsClient := client.GetSecretClient()

	// Get the secret 
	secretName := "bctl-" + os.Getenv("CLUSTER_NAME") + "-secret"
	secret, err := secretsClient.Get(context.Background(), secretName, metaV1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}

	// Return the secret
	return secret
}

func (client *ControlWebsocket) UpdateSecret(secret *coreV1.Secret) {
	// Build our client
	secretsClient := client.GetSecretClient()

	// Update the secret
	_, err := secretsClient.Update(context.Background(), secret, metaV1.UpdateOptions{})
	if err != nil {
		panic(err.Error())
	}
}

func (client *ControlWebsocket) GetSecretClient() coreV1Types.SecretInterface {
	// Create our api object
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Create our secrets client
	secretsClient := clientset.CoreV1().Secrets(os.Getenv("NAMESPACE"))

	return secretsClient
} 

func (client *ControlWebsocket) EncodeToBytes(p interface{}) []byte {
	// Ref: https://gist.github.com/SteveBate/042960baa7a4795c3565
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(p)
	if err != nil {
		log.Fatal(err)
	}
	return buf.Bytes()
}

func (client *ControlWebsocket) DecodeToSecretConfig(s []byte) SecretConfig {
	// Ref: https://gist.github.com/SteveBate/042960baa7a4795c3565
	p := SecretConfig{}
	dec := gob.NewDecoder(bytes.NewReader(s))
	err := dec.Decode(&p)
	if err != nil {
		log.Fatal(err)
	}
	return p
}