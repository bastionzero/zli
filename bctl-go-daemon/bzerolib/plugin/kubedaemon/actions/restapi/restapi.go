package restapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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

func (r *RestApiAction) InputMessageHandler(writer http.ResponseWriter, request *http.Request) (string, string, error) {
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
		return action, "", fmt.Errorf("Error building body")
	}

	// Build the action payload
	// If I'm returning this as an interface will it marshal correctly?
	payload := KubeRestApiActionPayload{
		Endpoint:  request.URL.String(),
		Headers:   headers,
		Method:    request.Method,
		Body:      string(bodyInBytes), // fix this
		RequestId: r.requestId,
	}

	payloadBytes, _ := json.Marshal(payload)
	return action, string(payloadBytes), nil // TODO: Make this bytes
}
