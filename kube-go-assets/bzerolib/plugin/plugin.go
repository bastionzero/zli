package plugin

import (
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

type PluginName string

const (
	Kube       PluginName = "kube"
	KubeDaemon PluginName = "kubedaemon"
)

type IPlugin interface {
	InputMessageHandler(action string, actionPayload []byte) (string, []byte, error)
	GetName() PluginName
	PushStreamInput(smessage smsg.StreamMessage) error
}

type ActionWrapper struct {
	Action        string
	ActionPayload []byte
}
