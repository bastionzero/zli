/*
This package defines the messages needed to unwrap and rewrap SignalR messages.
We've abstracted this wrapper so that we can move away from SignalR in the future,
and not have to reinvent our message structure.
*/
package message

type SignalRNegotiateResponse struct {
	NegotiateVersion int
	ConnectionId     string
}

// This is our SignalR wrapper, every message that comes in thru
// the data channel will be sent using SignalR, so we have to be
// able to unwrap and re-wrap it.  The AgentMessage is our generic
// message for everything we care about.
type SignalRWrapper struct {
	Target    string         `json:"target"` // hub name
	Type      int            `json:"type"`
	Arguments []AgentMessage `json:"arguments"`
}

// Hub name aka "Target" from "SignalRWrapper".  These are like API
// endpoint that the message will hit
type Hub string

const (
	RequestBastionToAgent  Hub = "RequestBastionToAgent"
	ResponseAgentToBastion Hub = "ResponseAgentToBastion"
)
