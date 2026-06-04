package ACS

type Config struct {
	//节点数
	N int
	//恶意节点数
	F int
	//节点ID
	ID uint64
	//所有节点ID
	Nodes []uint64
	//每个epoch提交的最大交易数量
	BatchSize int
}

type HBMessage struct {
	Epoch   uint64
	Payload interface{}
}

type BroadcastMessage struct {
	Payload interface{}
}

type AgreementMessage struct {
	Epoch   uint64
	Message interface{}
}
