package store

import (
	"learn_DumboNG/011-ACS/core"
	"log"

	"github.com/boltdb/bolt"
)

type BlockchainIterator struct {
	CurrentHash []byte
	DB          *bolt.DB
}

func (blockchainIterator *BlockchainIterator) Next() *core.Block {
	var block *core.Block
	err := blockchainIterator.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockTableName))
		if b != nil {
			currentBlockBytes := b.Get(blockchainIterator.CurrentHash)
			//获取到当前迭代器里面的currentHash所对应的区块
			block = core.DeserializeBlock(currentBlockBytes)
			//更新迭代器里面的currentHash
			blockchainIterator.CurrentHash = block.PrevBlockHash
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	return block
}
