package watch

// For "kube/watch/.." actions

const (
	WatchBufferSize = 1024 * 10
)

type KubeWatchActionPayload struct {
	Endpoint  string            `json:"endpoint"`
	Headers   map[string]string `json:"headers"`
	Method    string            `json:"method"`
	Body      string            `json:"body"`
	RequestId string            `json:"requestId"`
	End       bool              `json:"end"`
	LogId     string            `json:"logId"`
}

type KubeWatchHeadersPayload struct {
	Headers map[string][]string
}
