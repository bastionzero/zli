package error

type ErrorType string

const (
	// this is any error with the validation of the message itself
	// e.g. invalid signature, expired bzcert, wrong hpointer, etc.
	// The responding actions of any given error type should be the same
	KeysplittingValidationError ErrorType = "KeysplittingValidationError"

	// This error is essentially any error that comes from executing an
	// action aka if a file isn't found calling FUD, that error goes here.
	KeysplittingExecutionError ErrorType = "KeysplittingExecutionError"

	// Components such as datachannel, plugin, actions report their actions here.
	// Theoretically, there should be two kinds: any errors that come from
	// startup and any error independent of the message that arises during regular
	// functioning.
	ComponentStartupError    ErrorType = "ComponentStartupError"
	ComponentProcessingError ErrorType = "ComponentProcessingError"
)

type ErrorMessage struct {
	Type     string `json:"type"`
	Message  string `json:"message"`
	HPointer string `json:"hPointer"`
}
