package kubedaemon

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	plgn "bastionzero.com/bctl/v1/bzerolib/plugin"
	rest "bastionzero.com/bctl/v1/bzerolib/plugin/kubedaemon/actions/restapi"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

const (
	securityToken = "++++"
)

type KubeDaemonAction string

const (
	Exec    KubeDaemonAction = "exec"
	Log     KubeDaemonAction = "log"
	RestApi KubeDaemonAction = "restapi"
)

// Perhaps unnecessary but it is nice to make sure that each action is implementing a common function set
type IKubeDaemonAction interface {
	InputMessageHandler(writer http.ResponseWriter, request *http.Request) error
	PushKSResponse(actionWrapper plgn.ActionWrapper)
}

type KubeDaemonPlugin struct {
	localhostToken string
	daemonPort     string
	certPath       string
	keyPath        string

	// Input and output streams
	streamResponseChannel chan smsg.StreamMessage
	RequestChannel        chan plgn.ActionWrapper

	// To keep track of all current, ongoing actions
	actions map[int]IKubeDaemonAction
}

func NewKubeDaemonPlugin(localhostToken string, daemonPort string, certPath string, keyPath string) (*KubeDaemonPlugin, error) {
	plugin := KubeDaemonPlugin{
		localhostToken:        localhostToken,
		daemonPort:            daemonPort,
		certPath:              certPath,
		keyPath:               keyPath,
		streamResponseChannel: make(chan smsg.StreamMessage, 100),
		RequestChannel:        make(chan plgn.ActionWrapper, 100),
		actions:               make(map[int]IKubeDaemonAction),
	}

	go func() {
		for {
			select {
			case streamMessage := <-plugin.streamResponseChannel:
				plugin.handleStreamMessage(streamMessage)
			}
		}
	}()

	go func() {
		// Define our http handlers
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			plugin.rootCallback(w, r)
		})

		log.Fatal(http.ListenAndServeTLS(":"+plugin.daemonPort, plugin.certPath, plugin.keyPath, nil))
	}()

	return &plugin, nil
}

func (k *KubeDaemonPlugin) handleStreamMessage(smessage smsg.StreamMessage) error {
	return nil
}

func (k *KubeDaemonPlugin) PushStreamInput(smessage smsg.StreamMessage) error {
	k.streamResponseChannel <- smessage // maybe we don't need a middleman channel? eh, probably even if it's just a buffer
	return nil
}

func (k *KubeDaemonPlugin) GetName() plgn.PluginName {
	return plgn.KubeDaemon
}

func (k *KubeDaemonPlugin) InputMessageHandler(action string, actionPayload []byte) (string, []byte, error) {
	if len(actionPayload) > 0 {
		if x := strings.Split(action, "/"); len(x) <= 1 {
			return "", []byte{}, fmt.Errorf("Malformed action: %v", action)
		} else {
			var payload map[string]interface{}
			if err := json.Unmarshal([]byte(actionPayload), &payload); err != nil {
				return "", []byte{}, fmt.Errorf("Could not unmarshal actionPayload: %v", string(actionPayload))
			} else {
				// Json always unmarshals numbers as float64
				if id, ok := payload["requestId"].(float64); ok {
					log.Printf("Plugin recieved response for action with request ID: %v", id)
					if act, ok := k.actions[int(id)]; ok {
						wrappedAction := plgn.ActionWrapper{
							Action:        action,
							ActionPayload: actionPayload,
						}
						act.PushKSResponse(wrappedAction)
					} else {
						log.Printf("%+v", k.actions)
						return "", []byte{}, fmt.Errorf("Unknown Request ID")
					}
				} else {
					return "", []byte{}, fmt.Errorf("Action payload must include request ID")
				}
			}
		}
	}
	// TODO: check that plugin name is "kube"

	log.Printf("Waiting for input...")
	select {
	case actionMessage := <-k.RequestChannel:
		log.Printf("Received input from action: %v", actionMessage.Action)
		actionPayloadBytes, _ := json.Marshal(actionMessage.ActionPayload)
		return actionMessage.Action, actionPayloadBytes, nil
		// case <-time.After(time.Second * 10): // a better solution is to have a cancel channel
		// 	return "", "", fmt.Errorf("TIMEOUT!")
	}
}

func generateRequestId() int {
	// gotta mix up the see otherwise it always gives us the same number
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(10000) // We REALLY want to be using a uuid
}

func (k *KubeDaemonPlugin) rootCallback(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handling %s - %s\n", r.URL.Path, r.Method)

	// Trim off localhost token
	// TODO: Fix this
	k.localhostToken = strings.Replace(k.localhostToken, securityToken, "", -1) // ?

	// First verify our token and extract any commands if we can
	tokenToValidate := r.Header.Get("Authorization")

	// Remove the `Bearer `
	tokenToValidate = strings.Replace(tokenToValidate, "Bearer ", "", -1)

	// Validate the token
	tokensSplit := strings.Split(tokenToValidate, securityToken)
	if tokensSplit[0] != k.localhostToken {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/api") {
		id := generateRequestId()
		restAction, _ := rest.NewRestApiAction(id, k.RequestChannel)
		k.actions[id] = restAction
		log.Printf("Created Rest API action with id %v", id)
		if err := restAction.InputMessageHandler(w, r); err != nil {
			log.Printf("Error handling REST API call: %s", err.Error())
		}
		return
	}

	// if strings.Contains(r.URL.Path, "exec") {
	// 	handleExec.HandleExec(w, r, wsClient)
	// } else if strings.Contains(r.URL.Path, "log") {
	// 	handleLogs.HandleLogs(w, r, commandBeingRun, logId, wsClient)
	// } else {
	// 	handleREST.HandleREST(w, r, commandBeingRun, logId, wsClient)
	// }
}
