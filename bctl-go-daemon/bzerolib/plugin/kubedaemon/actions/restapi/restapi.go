package restapi

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"bastionzero.com/bctl/v1/bzerolib/keysplitting/hasher"
	kuberest "bastionzero.com/bctl/v1/bzerolib/plugin/kube/actions/restapi"
)

const (
	action = "kube/restapi"
)

type RestApiAction struct {
	requestId int
}

func NewRestApiAction(id int) (*RestApiAction, error) {
	return &RestApiAction{
		requestId: id,
	}, nil
}

func (r *RestApiAction) InputMessageHandler(writer http.ResponseWriter, request *http.Request) (string, []byte, error) {
	// First extract the headers out of the request
	headers := make(map[string]string)
	for name, values := range request.Header {
		for _, value := range values {
			headers[name] = value
		}
	}

	// Now extract the body
	bodyInBytes, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return action, []byte{}, fmt.Errorf("Error building body")
	}

	// Build the action payload
	// If I'm returning this as an interface will it marshal correctly?
	payload := kuberest.KubeRestApiActionPayload{
		Endpoint:  request.URL.String(),
		Headers:   headers,
		Method:    request.Method,
		Body:      string(bodyInBytes), // fix this
		RequestId: r.requestId,
	}

	payloadBytes, _ := hasher.SafeMarshal(payload)
	// payloadBytes, _ := json.Marshal(payload)
	return action, payloadBytes, nil // TODO: Make this bytes
}
