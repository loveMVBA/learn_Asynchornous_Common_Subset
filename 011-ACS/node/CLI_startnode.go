package node

import (
	"fmt"
	"learn_DumboNG/011-ACS/crypto"
	"learn_DumboNG/011-ACS/network"
	"os"
)

func (cli *CLI) startNode(nodeID string, minerAdd string) {
	if minerAdd != "" && crypto.IsValidForAddress([]byte(minerAdd)) == false {
		fmt.Println("Invalid miner address")
		os.Exit(1)
	}

	if minerAdd == "" {
		fmt.Printf("启动服务器：localhost:%s\n", nodeID)
	} else {
		fmt.Printf("启动服务器：localhost:%s，矿工地址：%s\n", nodeID, minerAdd)
	}
	network.StartServer(nodeID, minerAdd)
}
