package plugin

type PluginName string

const (
	Kube PluginName = "kube"
)

type IPlugin interface {
	InputMessageHandler(action string, actionPayload string) (interface{}, error)
	GetName() PluginName
}

type IAction interface {
	InputMessageHandler(action string, actionPayload string) (interface{}, error)
}
