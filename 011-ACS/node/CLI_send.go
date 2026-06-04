package node

import (
	"fmt"
	"learn_DumboNG/011-ACS/network"
	"learn_DumboNG/011-ACS/store"
)

// 转账
func (cli *CLI) send(from []string, to []string, amount []string, nodeID string, mineNow bool) {
	blockchain := store.BlockchainObject(nodeID)
	defer blockchain.DB.Close()

	txs := blockchain.CreateTransactions(from, to, amount, nodeID)
	network.SubmitTxs(nodeID, txs)
	fmt.Println("交易已提交到 ACS-BFT 网络，等待共识出块...")
}
