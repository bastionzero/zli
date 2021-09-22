package utils

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// Helper function to extract headers from a http request
func GetHeaders(headers http.Header) map[string]string {
	toReturn := make(map[string]string)
	for name, values := range headers {
		for _, value := range values {
			toReturn[name] = value
		}
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
