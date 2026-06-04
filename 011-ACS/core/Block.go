package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"learn_DumboNG/011-ACS/crypto"
	"log"
)

type Block struct {
	//1. 区块高度
	Height int64
	//2. 上一个区块hash
	PrevBlockHash []byte
	//3. 交易数据
	Txs []*Transaction
	//4. 时间戳。ACS-BFT 中需要所有节点确定性构造同一区块，这里用高度作为确定性时间戳。
	Timestamp int64
	//5。 当前区块hash
	Hash []byte
	//6. Nonce。ACS-BFT 不挖矿，保留字段但固定为 0。
	Nonce int64
}

// 1. 创建新的区块
func NewBlock(txs []*Transaction, height int64, prevBlockHash []byte) *Block {
	block := &Block{height, prevBlockHash, txs, height, nil, 0}
	block.Hash = block.CalculateHash()
	return block
}

// 2. 生成创世区块
func CreateGenesisBlock(txs []*Transaction) *Block {
	return NewBlock(txs, 1, make([]byte, 32))
}

func (block *Block) CalculateHash() []byte {
	data := bytes.Join(
		[][]byte{
			block.PrevBlockHash,
			block.HashTransactions(),
			crypto.IntTOHex(block.Timestamp),
			crypto.IntTOHex(block.Height),
		},
		[]byte{},
	)
	hash := sha256.Sum256(data)
	return hash[:]
}

// 3.将区块序列化成字节数组
func (block *Block) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(block)
	if err != nil {
		log.Panic(err)
	}
	return result.Bytes()
}

func DeserializeBlock(blockBytes []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(blockBytes))
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}
	return &block
}

// 4.将Txs转换成字节数组[]byte
func (block *Block) HashTransactions() []byte {
	var transactions [][]byte
	for _, tx := range block.Txs {
		transactions = append(transactions, tx.Serialize())
	}
	mTree := NewMerkleTree(transactions)
	return mTree.RootNode.Data
}
