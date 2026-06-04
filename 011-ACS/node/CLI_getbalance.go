package node

import (
	"fmt"
	"learn_DumboNG/011-ACS/store"
	"os"
)

func (cli *CLI) getBalance(address string, nodeID string) {
	if store.DBExists(nodeID) == false {
		fmt.Println("数据库不存在.....")
		os.Exit(1)
	}
	blockchain := store.BlockchainObject(nodeID)
	defer blockchain.DB.Close()

	fmt.Println("地址： ", address)
	amount := blockchain.GetBalance(address)
	fmt.Printf("%s 的余额一共有 %d 个Token\n", address, amount)
}
