package node

import (
	"fmt"
	"learn_DumboNG/011-ACS/crypto"
)

func (cli *CLI) CreateWallet(nodeID string) {
	wallets, _ := crypto.NewWallets(nodeID)
	wallets.CreateNewWallet(nodeID)
	fmt.Println(len(wallets.WalletsMap))
}
