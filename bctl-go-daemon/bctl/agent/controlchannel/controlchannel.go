package controlchannel

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sync"

	"bastionzero.com/bctl/v1/bctl/agent/vault"
	wsmsg "bastionzero.com/bctl/v1/bzerolib/channels/message"
	ws "bastionzero.com/bctl/v1/bzerolib/channels/websocket"
	lggr "bastionzero.com/bctl/v1/bzerolib/logger"
	"golang.org/x/crypto/sha3"

	ed "crypto/ed25519"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	hubEndpoint       = "/api/v1/hub/kube-control"
	registerEndpoint  = "/api/v1/kube/register-agent"
	challengeEndpoint = "/api/v1/kube/get-challenge"
	autoReconnect     = true
)

type ControlChannel struct {
	websocket *ws.Websocket
	logger    *lggr.Logger

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

	logger := lggr.NewLogger(lggr.Controlchannel, lggr.Debug)
	subLogger := logger.GetWebsocketSubLogger()

	// Populate keys if they haven't been generated already
	config, err := newAgent(logger, serviceUrl, activationToken, agentVersion, orgId, environmentId, clusterName)
	if err != nil {
		logger.Error(err)
		return &ControlChannel{}, err
	}

	solvedChallenge, err := getAndSolveChallenge(orgId, clusterName, serviceUrl, config.Data.PrivateKey)
	if err != nil {
		logger.Error(err)
		return &ControlChannel{}, err
	}

	// Create our headers and params, headers are empty
	headers := make(map[string]string)

	// Make and add our params
	params := map[string]string{
		"solved_challange": solvedChallenge,
		"public_key":       config.Data.PublicKey,
		"agent_version":    agentVersion,

		// Why do we need these?  Can we remove them?
		"org_id":         orgId,
		"cluster_name":   clusterName,
		"environment_id": environmentId,
	}

	msg := fmt.Sprintf("{serviceURL: %v, hubEndpoint: %v, params: %v, headers: %v}", serviceUrl, hubEndpoint, params, headers)
	logger.Info(msg)

	wsClient, err := ws.NewWebsocket(subLogger, serviceUrl, hubEndpoint, params, headers, targetSelectHandler, autoReconnect, true)
	if err != nil {
		return &ControlChannel{}, err
	}

	control := ControlChannel{
		websocket:          wsClient,
		NewDatachannelChan: make(chan NewDatachannelMessage),
		logger:             logger,
	}

	// Set up our handler to deal with incoming messages
	go func() {
		for {
			select {
			case <-control.websocket.DoneChannel:
				control.logger.Info("Websocket has been closed, closing controlchannel")
				return
			case agentMessage := <-control.websocket.InputChannel:
				if err := control.Receive(agentMessage); err != nil {
					control.logger.Error(err)
					return
				}
			}
		}
	}()
	return &control, nil
}

func (c *ControlChannel) Receive(agentMessage wsmsg.AgentMessage) error {
	switch wsmsg.MessageType(agentMessage.MessageType) {
	case wsmsg.NewDatachannel:
		var dataMessage NewDatachannelMessage
		if err := json.Unmarshal(agentMessage.MessagePayload, &dataMessage); err != nil {
			return fmt.Errorf("error unmarshalling new controlchannel request: %v", err.Error())
		} else {
			c.NewDatachannelChan <- dataMessage
		}
	case wsmsg.HealthCheck:
		if msg, err := healthCheck(); err != nil {
			return err
		} else {
			c.websocket.OutputChannel <- wsmsg.AgentMessage{
				MessageType:    string(wsmsg.HealthCheck),
				SchemaVersion:  wsmsg.SchemaVersion,
				MessagePayload: msg,
			}
		}
	}
	return nil
}

func getAndSolveChallenge(orgId string, clusterName string, serviceUrl string, privateKey string) (string, error) {
	// Get Challenge
	challengeRequest := GetChallengeMessage{
		OrgId:       orgId,
		ClusterName: clusterName,
	}

	challengeJson, err := json.Marshal(challengeRequest)
	if err != nil {
		return "", errors.New("error marshalling register data")
	}

	// Make our POST request
	response, err := http.Post(
		"https://"+serviceUrl+challengeEndpoint,
		"application/json",
		bytes.NewBuffer(challengeJson))
	if err != nil || response.StatusCode != http.StatusOK {
		rerr := fmt.Errorf("error making post request to challenge agent. Error: %v. Response: %v", err, response)
		return "", rerr
	}
	defer response.Body.Close()

	// Extract the challenge
	responseDecoded := GetChallengeResponse{}
	json.NewDecoder(response.Body).Decode(&responseDecoded)

	// Solve Challenge
	return SignChallenge(privateKey, responseDecoded.Challenge)
}

// TODO: make a bzerolib signing function
func SignChallenge(privateKey string, challenge string) (string, error) {
	keyBytes, _ := base64.StdEncoding.DecodeString(privateKey)
	if len(keyBytes) != 64 {
		return "", fmt.Errorf("invalid private key length: %v", len(keyBytes))
	}
	privkey := ed.PrivateKey(keyBytes)

	hashBits := sha3.Sum256([]byte(challenge))

	sig := ed.Sign(privkey, hashBits[:])

	// Convert the signature to base64 string
	sigBase64 := base64.StdEncoding.EncodeToString(sig)

	return sigBase64, nil
}

func healthCheck() ([]byte, error) {
	// Also let bastion know a list of valid cluster roles
	// TODO: break out extracting the list of valid cluster roles
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

func newAgent(logger *lggr.Logger, serviceUrl string, activationToken string, agentVersion string, orgId string, environmentId string, clusterName string) (*vault.Vault, error) {
	config, _ := vault.LoadVault()

	// Check if vault is empty, if so generate a private, public key pair
	if config.IsEmpty() {
		logger.Info("Creating new agent secret")

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
				return nil, fmt.Errorf("error saving vault: %s", err)
			}

			// Register with Bastion
			logger.Info("Registering agent with Bastion")
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
				msg := fmt.Errorf("error marshalling registration data: %s", err)
				return nil, msg
			}

			// Make our POST request
			response, err := http.Post("https://"+serviceUrl+registerEndpoint, "application/json",
				bytes.NewBuffer(registerJson))
			if err != nil || response.StatusCode != http.StatusOK {
				rerr := fmt.Errorf("error making post request to register agent. Error: %s. Response: %v", err, response)
				return nil, rerr
			}
		}
	} else {
		// If the vault isn't empty, don't do anything
		logger.Info("Found Previous config data")
	}
	return config, nil
}
