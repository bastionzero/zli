package keysplitting

type IKeysplitting interface {
	ProcessSyn()
	ProcessData()
}

type Keysplitting struct {
	HPointer         string
	ExpectedHPointer string
	bzeCerts         map[string]map[string]interface{}
}
