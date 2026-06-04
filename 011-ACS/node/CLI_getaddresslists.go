package node

import (
	"fmt"
	"learn_DumboNG/011-ACS/crypto"
)

func (cli *CLI) addressLists(nodeID string) {
	fmt.Println("所有钱包地址如下:")
	wallets, _ := crypto.NewWallets(nodeID)
	for address := range wallets.WalletsMap {
		fmt.Println(address)
	}
}
