package node

import "learn_DumboNG/011-ACS/store"

func (cli *CLI) printchain(nodeID string) {
	blockchain := store.BlockchainObject(nodeID)
	defer blockchain.DB.Close()
	blockchain.Printchain()
}
