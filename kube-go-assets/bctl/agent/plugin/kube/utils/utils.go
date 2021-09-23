package utils

import "fmt"

func ValidateRequestId(requestIdPassed string, requestIdSaved string) error {
	if requestIdPassed != requestIdSaved {
		rerr := fmt.Errorf("invalid request ID passed")
		return rerr
	}
	return nil
}
