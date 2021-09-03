package controlchannel

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sync"

	"bastionzero.com/bctl/v1/bctl/agent/vault"
	wsmsg "bastionzero.com/bctl/v1/bzerolib/channels/message"
	ws "bastionzero.com/bctl/v1/bzerolib/channels/websocket"
	lggr "bastionzero.com/bctl/v1/bzerolib/logger"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	hubEndpoint       = "/api/v1/hub/kube-control"
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
func NewControlChannel(logger *lggr.Logger,
	serviceUrl string,
	activationToken string,
	orgId string,
	clusterName string,
	environmentId string,
	agentVersion string,
	targetSelectHandler func(msg wsmsg.AgentMessage) (string, error)) (*ControlChannel, error) {

	subLogger := logger.GetWebsocketLogger()

	// Load in our saved config
	config, _ := vault.LoadVault()

	// Create our headers and params, headers are empty
	headers := make(map[string]string)

	// Make and add our params
	params := map[string]string{
		"public_key":    config.Data.PublicKey,
		"agent_version": agentVersion,

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
