package network

// 存储各个节点的全局变量。
// ACS-BFT 默认使用 4 个验证节点，允许 1 个拜占庭节点。
var knowNodes = []string{"localhost:3000", "localhost:3001", "localhost:3002", "localhost:3003"}
var nodeAddress string
var transactionArray [][]byte
var consensusRuntime *ACSRuntime
