package store

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"learn_DumboNG/011-ACS/core"
	"learn_DumboNG/011-ACS/crypto"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
)

const dbName = "blockchain_%s.db"
const blockTableName = "blocks"

type Blockchain struct {
	Tip []byte
	DB  *bolt.DB
}

func (bc *Blockchain) AddBlock(block *core.Block) error {
	err := bc.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockTableName))
		if b != nil {
			blockExist := b.Get(block.Hash)
			if blockExist != nil {
				return nil
			}

			err := b.Put(block.Hash, block.Serialize())
			if err != nil {
				return err
			}

			blockHash := b.Get([]byte("l"))
			if len(blockHash) == 0 {
				b.Put([]byte("l"), block.Hash)
				bc.Tip = block.Hash
				return nil
			}

			blockBytes := b.Get(blockHash)
			blockInDB := core.DeserializeBlock(blockBytes)

			if blockInDB.Height < block.Height {
				b.Put([]byte("l"), block.Hash)
				bc.Tip = block.Hash
			}
		}
		return nil
	})

	return err
}

func (bc *Blockchain) GetBlock(blockHash []byte) (*core.Block, error) {
	var block *core.Block

	err := bc.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockTableName))
		if b != nil {
			blockBytes := b.Get(blockHash)
			if blockBytes == nil {
				return fmt.Errorf("block %x not found", blockHash)
			}
			block = core.DeserializeBlock(blockBytes)
		}
		return nil
	})

	return block, err
}

func (bc *Blockchain) GetBestHeight() int64 {
	if bc == nil || len(bc.Tip) == 0 {
		return 0
	}
	block := bc.Iterator().Next()

	return block.Height
}

func (bc *Blockchain) GetBlockHashes() [][]byte {
	var blockHashes [][]byte
	if bc == nil || len(bc.Tip) == 0 {
		return blockHashes
	}

	blockIterator := bc.Iterator()

	for {
		block := blockIterator.Next()
		blockHashes = append(blockHashes, block.Hash)

		var hashInt big.Int
		hashInt.SetBytes(block.PrevBlockHash)

		if hashInt.Cmp(big.NewInt(0)) == 0 {
			break
		}
	}

	return blockHashes
}

// 14.查找UTXO映射 [string]*core.TXOutputs
func (blc *Blockchain) FindUTXOMap() map[string]*core.TXOutputs {

	blcIterator := blc.Iterator()
	spendableUTXOMap := make(map[string][]*core.TXInput)
	utxoMaps := make(map[string]*core.TXOutputs)

	for {
		block := blcIterator.Next()
		for i := len(block.Txs) - 1; i >= 0; i-- {
			txOutputs := &core.TXOutputs{[]*core.UTXO{}}
			tx := block.Txs[i]

			if tx.IsCoinbaseTransaction() == false {
				for _, txInput := range tx.Vins {
					txHash := hex.EncodeToString(txInput.TxHash)
					spendableUTXOMap[txHash] = append(spendableUTXOMap[txHash], txInput)
				}
			}

			if tx.IsCoinbaseTransaction() {

			}
			txHash := hex.EncodeToString(tx.TxHash)

		WorkOutLoop:
			for index, out := range tx.Vouts {
				txInputs := spendableUTXOMap[txHash]
				if len(txInputs) > 0 {
					isSpent := false
					for _, in := range txInputs {
						outPublicKey := out.Ripemd160Hash
						inPublicKey := in.PublicKey
						if bytes.Equal(outPublicKey, crypto.Ripemd160Hash(inPublicKey)) {
							if index == in.Vout {
								isSpent = true
								continue WorkOutLoop
							}
						}
					}
					if isSpent == false {
						utxo := &core.UTXO{tx.TxHash, index, out}
						txOutputs.UTXOS = append(txOutputs.UTXOS, utxo)
					}
				} else {
					utxo := &core.UTXO{tx.TxHash, index, out}
					txOutputs.UTXOS = append(txOutputs.UTXOS, utxo)
				}
			}

			utxoMaps[txHash] = txOutputs
		}

		var hashInt big.Int
		hashInt.SetBytes(block.PrevBlockHash)

		if hashInt.Cmp(big.NewInt(0)) == 0 {
			break
		}
	}

	return utxoMaps

}

// 13.验证交易签名
func (bc *Blockchain) VerifyTransaction(tx *core.Transaction, txs []*core.Transaction) bool {
	if tx.IsCoinbaseTransaction() {
		return true
	}

	prevTXs := make(map[string]core.Transaction)

	for _, vin := range tx.Vins {
		prevTX, err := bc.FindTransaction(vin.TxHash, txs)
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.TxHash)] = prevTX
	}

	return tx.Verify(prevTXs)
}

// 12.签名交易
func (blockchain *Blockchain) SignTransaction(tx *core.Transaction, privateKey ecdsa.PrivateKey, txs []*core.Transaction) {

	if tx.IsCoinbaseTransaction() {
		return
	}

	prevTXs := make(map[string]core.Transaction)

	for _, vin := range tx.Vins {
		prevTX, err := blockchain.FindTransaction(vin.TxHash, txs)
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.TxHash)] = prevTX
	}

	tx.Sign(privateKey, prevTXs)
}

// 11.查找交易
func (bc *Blockchain) FindTransaction(ID []byte, txs []*core.Transaction) (core.Transaction, error) {

	for _, tx := range txs {
		if bytes.Equal(tx.TxHash, ID) {
			return *tx, nil
		}
	}

	bci := bc.Iterator()

	for {
		block := bci.Next()
		for _, tx := range block.Txs {
			if bytes.Equal(tx.TxHash, ID) {
				return *tx, nil
			}
		}

		var hashInt big.Int
		hashInt.SetBytes(block.PrevBlockHash)
		if big.NewInt(0).Cmp(&hashInt) == 0 {
			break
		}

	}

	return core.Transaction{}, fmt.Errorf("transaction %x not found", ID)
}

// 10.转账时查找可用的UTXO
func (blockchain *Blockchain) FindSpendableUTXOs(from string, amount int, txs []*core.Transaction) (int64, map[string][]int) {

	//1.获取所有的UTXO
	utxos := blockchain.UnUTXOs(from, txs)

	//2.遍历所有的UTXO，找到可用的UTXO
	spendableUTXO := make(map[string][]int)
	var value int64
	for _, utxo := range utxos {
		value = value + utxo.Output.Value

		hash := hex.EncodeToString(utxo.TxHash)
		spendableUTXO[hash] = append(spendableUTXO[hash], utxo.Index)

		if value >= int64(amount) {
			break
		}
	}
	if value < int64(amount) {
		fmt.Println("&s 的余额不足", from)
		os.Exit(1)
	}
	return value, spendableUTXO
}

// 9.查询余额
func (blockchain *Blockchain) GetBalance(address string) int64 {
	utxos := blockchain.UnUTXOs(address, []*core.Transaction{})
	var amount int64
	for _, utxo := range utxos {
		amount += utxo.Output.Value
	}
	return amount
}

// 8.如果一个地址对应的TXOutput未花费，添加到数组中并返回该地址对应的所有未花费的Transaction
func (blockchain *Blockchain) UnUTXOs(address string, txs []*core.Transaction) []*core.UTXO {

	var unUTXOs []*core.UTXO
	spentTXOutputs := make(map[string][]int) //我要一个字典 {hash: [index1, index2, ...]}

	for _, tx := range txs {
		if tx.IsCoinbaseTransaction() == false {
			for _, in := range tx.Vins {
				publicKeyHash := crypto.Base58Decode([]byte(address))
				ripemd160Hash := publicKeyHash[1 : len(publicKeyHash)-4]

				if in.UnLockRipemd160(ripemd160Hash) {
					key := hex.EncodeToString(in.TxHash)
					spentTXOutputs[key] = append(spentTXOutputs[key], in.Vout)
				}
			}
		}
	}

	for _, tx := range txs {
	Work1:
		for index, out := range tx.Vouts {
			if out.UnLockScriptPubKeyWithAddress(address) {
				if len(spentTXOutputs) == 0 {
					utxo := &core.UTXO{
						TxHash: tx.TxHash,
						Index:  index,
						Output: out,
					}
					unUTXOs = append(unUTXOs, utxo)
				} else {
					for hash, indexArray := range spentTXOutputs {
						txHashStr := hex.EncodeToString(tx.TxHash)
						if hash == txHashStr {
							var isUnspentUTXO bool
							for _, outIndex := range indexArray {
								if index == outIndex {
									isUnspentUTXO = true
									continue Work1
								}
								if isUnspentUTXO == false {
									utxo := &core.UTXO{
										TxHash: tx.TxHash,
										Index:  index,
										Output: out,
									}
									unUTXOs = append(unUTXOs, utxo)
								}
							}

						} else {
							utxo := &core.UTXO{
								TxHash: tx.TxHash,
								Index:  index,
								Output: out,
							}
							unUTXOs = append(unUTXOs, utxo)
						}
					}
				}
			}

		}
	}

	blockIterator := blockchain.Iterator()

	for {
		block := blockIterator.Next()
		fmt.Println(block)
		fmt.Println()

		for i := len(block.Txs) - 1; i >= 0; i-- {
			tx := block.Txs[i]
			// txHash

			//Vins
			if tx.IsCoinbaseTransaction() == false {
				for _, in := range tx.Vins {
					publicKeyHash := crypto.Base58Decode([]byte(address))
					ripemd160Hash := publicKeyHash[1 : len(publicKeyHash)-4]

					if in.UnLockRipemd160(ripemd160Hash) {
						key := hex.EncodeToString(in.TxHash)
						spentTXOutputs[key] = append(spentTXOutputs[key], in.Vout)
					}
				}
			}
			//Vouts
		work:
			for index, out := range tx.Vouts {
				if out.UnLockScriptPubKeyWithAddress(address) {
					if len(spentTXOutputs) != 0 {
						var isSpentUTXO bool
						for txHash, indexArray := range spentTXOutputs {
							for _, i := range indexArray {
								if index == i && txHash == hex.EncodeToString(tx.TxHash) {
									isSpentUTXO = true
									continue work
								}
							}
						}
						if isSpentUTXO == false {
							utxo := &core.UTXO{
								TxHash: tx.TxHash,
								Index:  index,
								Output: out,
							}
							unUTXOs = append(unUTXOs, utxo)
						}
					} else {
						utxo := &core.UTXO{
							TxHash: tx.TxHash,
							Index:  index,
							Output: out,
						}
						unUTXOs = append(unUTXOs, utxo)
					}
				}
			}

		}

		var hashInt big.Int
		hashInt.SetBytes(block.PrevBlockHash)
		if hashInt.Cmp(big.NewInt(0)) == 0 {
			break
		}
	}

	return unUTXOs
}

// CreateTransactions 根据转账参数创建已签名交易，但不直接出块。
func (blockchain *Blockchain) CreateTransactions(from []string, to []string, amount []string, nodeID string) []*core.Transaction {
	utxoSet := &UTXOSet{blockchain}
	var txs []*core.Transaction
	for index, address := range from {
		value, _ := strconv.Atoi(amount[index])
		tx := core.NewSimpleTransaction(address, to[index], int64(value), utxoSet, txs, nodeID)
		txs = append(txs, tx)
	}
	return txs
}

// BuildBlock 基于已经达成 ACS 共识的交易集合，确定性构造新区块。
func (blockchain *Blockchain) BuildBlock(txs []*core.Transaction) *core.Block {
	var block *core.Block
	blockchain.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockTableName))
		if b != nil {
			hash := b.Get([]byte("l"))
			blockBytes := b.Get(hash)
			block = core.DeserializeBlock(blockBytes)
		}
		return nil
	})
	return core.NewBlock(txs, block.Height+1, block.Hash)
}

// CommitBlock 提交 ACS 已决定的区块，并更新 UTXO 集。
func (blockchain *Blockchain) CommitBlock(block *core.Block) {
	err := blockchain.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockTableName))
		if b != nil {
			b.Put(block.Hash, block.Serialize())
			b.Put([]byte("l"), block.Hash)
			blockchain.Tip = block.Hash
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	utxoSet := &UTXOSet{blockchain}
	utxoSet.ResetUTXOSet()
}

// MineNewBlock 保留兼容入口；ACS 版本中不再挖矿，只创建并提交本地确定性区块。
func (blockchain *Blockchain) MineNewBlock(from []string, to []string, amount []string, nodeID string) {
	txs := blockchain.CreateTransactions(from, to, amount, nodeID)
	block := blockchain.BuildBlock(txs)
	blockchain.CommitBlock(block)
}

// 6.返回区块链对象的方法
func BlockchainObject(nodeID string) *Blockchain {
	dbFile := fmt.Sprintf(dbName, nodeID)

	if DBExists(nodeID) == false {
		fmt.Println("数据库不存在...")
		os.Exit(1)
	}

	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	var tip []byte

	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(blockTableName))
		if err != nil {
			return err
		}
		//读取最新区块的hash
		tip = b.Get([]byte("l"))
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return &Blockchain{tip, db}
}

// 4.迭代器方法
func (blockchain *Blockchain) Iterator() *BlockchainIterator {

	return &BlockchainIterator{blockchain.Tip, blockchain.DB}
}

// 3.遍历输出所有区块的信息
func (blc *Blockchain) Printchain() {

	blockchainIterator := blc.Iterator()

	for {
		block := blockchainIterator.Next()

		fmt.Printf("Height: %d\n", block.Height)
		fmt.Printf("PrevBlockHash: %x\n", block.PrevBlockHash)
		fmt.Printf("Timestamp: %s\n", time.Unix(block.Timestamp, 0).Format("2006-01-02 03:04:05 PM"))
		fmt.Printf("Hash: %x\n", block.Hash)
		fmt.Printf("Nonce: %d\n", block.Nonce)
		fmt.Println("Txs :")
		for _, tx := range block.Txs {
			fmt.Printf("%x\n", tx.TxHash)
			fmt.Println("Vins :")
			for _, in := range tx.Vins {
				fmt.Printf("%x\n", in.TxHash)
				fmt.Printf("%d\n", in.Vout)
				fmt.Printf("%s\n", in.PublicKey)
			}

			fmt.Println("Vouts:")
			for _, out := range tx.Vouts {
				fmt.Println(out.Value)
				fmt.Println(out.Ripemd160Hash)
			}

		}

		fmt.Println()

		var hashInt big.Int
		hashInt.SetBytes(block.PrevBlockHash)
		if big.NewInt(0).Cmp(&hashInt) == 0 {
			break
		}
	}

}

// 2.增加区块到区块链里面
func (blc *Blockchain) AddBlocktoBlockchain(txs []*core.Transaction) {

	err := blc.DB.Update(func(tx *bolt.Tx) error {

		//1， 获取表
		b := tx.Bucket([]byte(blockTableName))

		//2. 创建新区块
		if b != nil {
			//获取最新区块
			blockBytes := b.Get(blc.Tip)
			block := core.DeserializeBlock(blockBytes)

			//3. 将区块序列化并存储到数据库
			newBlock := core.NewBlock(txs, block.Height+1, block.Hash)
			err := b.Put(newBlock.Hash, newBlock.Serialize())
			if err != nil {
				log.Panic(err)
			}

			//4. 更新数据库里面的“l"对应的hash
			err = b.Put([]byte("l"), newBlock.Hash)
			if err != nil {
				log.Panic(err)
			}

			//5. 更新blockchain的Tip
			blc.Tip = newBlock.Hash
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}

// 1. 创建带有创世区块的区块链
// 判断数据库是否存在
func DBExists(nodeID string) bool {
	dbFile := fmt.Sprintf(dbName, nodeID)

	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}

func InitEmptyBlockchainDB(nodeID string) {
	dbFile := fmt.Sprintf(dbName, nodeID)
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(blockTableName))
		return err
	})
	if err != nil {
		log.Panic(err)
	}
}

func CreateBlockchainWithGenesisBlock(address string, nodeID string) *Blockchain {
	//判断数据库是否存在
	if DBExists(nodeID) {
		fmt.Println("创世区块已存在......")
		os.Exit(1)
	}

	fmt.Println("正在创建创世区块...")

	//创建并打开数据库
	dbFile := fmt.Sprintf(dbName, nodeID)
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	var genesisHash []byte
	err = db.Update(func(tx *bolt.Tx) error {

		b, err := tx.CreateBucket([]byte(blockTableName))
		if err != nil {
			log.Panic(err)
		}

		if b != nil {
			txCoinbase := core.NewCoinbaseTransaction(address)
			//创建创世区块
			genesisBlock := core.CreateGenesisBlock([]*core.Transaction{txCoinbase})
			//将创世区块存储到表中
			err := b.Put(genesisBlock.Hash, genesisBlock.Serialize())
			if err != nil {
				log.Panic(err)
			}
			//存储最新的区块的hash
			err = b.Put([]byte("l"), genesisBlock.Hash)
			if err != nil {
				log.Panic(err)
			}

			genesisHash = genesisBlock.Hash
		}

		return nil
	})

	return &Blockchain{Tip: genesisHash, DB: db}
}
