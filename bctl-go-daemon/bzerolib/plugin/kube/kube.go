package kube

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	plgn "bastionzero.com/bctl/v1/bzerolib/plugin"
	rest "bastionzero.com/bctl/v1/bzerolib/plugin/kube/actions/restapi"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

const (
	impersonateGroup = "system:authenticated"
)

type IKubeAction interface {
	InputMessageHandler(action string, actionPayload string) (string, []byte, error)
}

type KubeAction string

const (
	Exec    KubeAction = "exec"
	Log     KubeAction = "log"
	RestApi KubeAction = "restapi"
)

type KubePlugin struct {
	role                string
	streamOutputChannel chan smsg.StreamMessage
	serviceAccountToken string
	kubeHost            string
	runningActions      []IKubeAction // need something like this for streams and multiple tabs when running exec & logs
}

func NewPlugin(ch chan smsg.StreamMessage, role string) plgn.IPlugin {
	// First load in our Kube variables
	// TODO: Where should we save this, in the class? is this the best way to do this?
	// TODO: Also we should be able to drop this req, and just load `IN CLUSTER CONFIG`
	serviceAccountTokenPath := os.Getenv("KUBERNETES_SERVICE_ACCOUNT_TOKEN_PATH")
	serviceAccountTokenBytes, _ := ioutil.ReadFile(serviceAccountTokenPath)
	// TODO: Check for error
	serviceAccountToken := string(serviceAccountTokenBytes)
	kubeHost := "https://" + os.Getenv("KUBERNETES_SERVICE_HOST")

	return &KubePlugin{
		role:                role,
		streamOutputChannel: ch,
		serviceAccountToken: serviceAccountToken,
		kubeHost:            kubeHost,
	}
}

func (k *KubePlugin) GetName() plgn.PluginName {
	return plgn.Kube
}

func (k *KubePlugin) PushStreamInput(smessage smsg.StreamMessage) error {
	return fmt.Errorf("")
}

func (k *KubePlugin) InputMessageHandler(action string, actionPayload []byte) (string, []byte, error) {
	log.Printf("Plugin received Data message with %v action", action)
	x := strings.Split(action, "/")
	if len(x) < 2 {
		return "", []byte{}, fmt.Errorf("Malformed action: %s", action)
	}
	kubeAction := x[1]

	switch KubeAction(kubeAction) {
	case RestApi:
		a, _ := rest.NewRestApiAction(k.serviceAccountToken, k.kubeHost, impersonateGroup, k.role)
		return a.InputMessageHandler(action, actionPayload)
	case Exec:
		break
	case Log:
		break
	default:
		return "", []byte{}, fmt.Errorf("Unhandled action")
	}

	return "", []byte{}, nil
}
