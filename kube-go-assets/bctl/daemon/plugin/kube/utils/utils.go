package utils

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// Helper function to extract headers from a http request
func GetHeaders(headers http.Header) map[string][]string {
	toReturn := make(map[string][]string)
	for name, values := range headers {
		toReturn[name] = values
	}
	return toReturn
}

// Helper function to extract the body of a http request
func GetBodyBytes(body io.ReadCloser) ([]byte, error) {
	bodyInBytes, err := ioutil.ReadAll(body)
	if err != nil {
		rerr := fmt.Errorf("error building body: %s", err)
		return nil, rerr
	}
	return bodyInBytes, nil
}

func WriteToHttpRequest(contentBytes []byte, writer http.ResponseWriter) error {
	src := bytes.NewReader(contentBytes)
	_, err := io.Copy(writer, src)
	if err != nil {
		rerr := fmt.Errorf("error streaming data to kubectl: %s", err)
		return rerr
	}
	// This is required to flush the data to the client
	flush, ok := writer.(http.Flusher)
	if ok {
		flush.Flush()
	}
	return nil
}

func IsQueryParamPresent(request *http.Request, paramArg string) bool {
	// Get the param from the query
	param, ok := request.URL.Query()[paramArg]

	// First check if we got any query returned
	if !ok || len(param[0]) < 1 {
		return false
	}

	// Now check if param is a valid value
	if param[0] == "true" || param[0] == "1" {
		return true
	}

	// Else return false
	return false
}
