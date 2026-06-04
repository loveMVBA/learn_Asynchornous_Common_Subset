package store

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"learn_DumboNG/011-ACS/core"
	"learn_DumboNG/011-ACS/crypto"
	"log"

	"github.com/boltdb/bolt"
)

// 遍历整个数据库，读取所有未花费的UTXO，存储所有UTXO到数据库
type UTXOSet struct {
	Blockchain *Blockchain
}

const utxoTableName = "utxoTableName"

// 重置数据库table
func (utxoSet *UTXOSet) ResetUTXOSet() {
	err := utxoSet.Blockchain.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoTableName))

		if b != nil {
			err := tx.DeleteBucket([]byte(utxoTableName))
			if err != nil {
				log.Panic(err)
			}
		}

		b, _ = tx.CreateBucket([]byte(utxoTableName))
		if b != nil {
			txOutputsMap := utxoSet.Blockchain.FindUTXOMap()
			for keyHash, outs := range txOutputsMap {
				txHash, _ := hex.DecodeString(keyHash)
				b.Put(txHash, outs.Serialize())
			}

		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}

func (utxoSet *UTXOSet) findUTXOForAddress(address string) []*core.UTXO {
	var utxos []*core.UTXO

	utxoSet.Blockchain.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoTableName))
		c := b.Cursor()
		if b != nil {
			for k, v := c.First(); k != nil; k, v = c.Next() {
				txOutputs := core.DeserializeTXOutputs(v)
				for _, utxo := range txOutputs.UTXOS {
					if utxo.Output.UnLockScriptPubKeyWithAddress(address) {
						utxos = append(utxos, utxo)
					}
				}
			}
		}
		return nil
	})

	return utxos
}

func (utxoSet *UTXOSet) GetBalance(address string) int64 {
	UTXOS := utxoSet.findUTXOForAddress(address)
	var amount int64
	for _, utxo := range UTXOS {
		amount += utxo.Output.Value
	}
	return amount
}

func (utxoSet *UTXOSet) FindUnPackageSpendableUTXOs(from string, txs []*core.Transaction) []*core.UTXO {
	var unUTXOs []*core.UTXO
	spentTXOutputs := make(map[string][]int) //我要一个字典 {hash: [index1, index2, ...]}

	for _, tx := range txs {
		if tx.IsCoinbaseTransaction() == false {
			for _, in := range tx.Vins {
				publicKeyHash := crypto.Base58Decode([]byte(from))
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
			if out.UnLockScriptPubKeyWithAddress(from) {
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
	return unUTXOs
}

func (utxoSet *UTXOSet) SignTransaction(tx *core.Transaction, privateKey ecdsa.PrivateKey, txs []*core.Transaction) {
	utxoSet.Blockchain.SignTransaction(tx, privateKey, txs)
}

func (utxoSet *UTXOSet) FindSpendableUTXOs(from string, amount int64, txs []*core.Transaction) (int64, map[string][]int) {

	unPackageUTXOS := utxoSet.FindUnPackageSpendableUTXOs(from, txs)

	spentableUTXOs := make(map[string][]int)
	var money int64 = 0
	for _, utxo := range unPackageUTXOS {
		money += utxo.Output.Value
		txHash := hex.EncodeToString(utxo.TxHash)
		spentableUTXOs[txHash] = append(spentableUTXOs[txHash], utxo.Index)
		if money >= amount {
			return money, spentableUTXOs
		}
	}

	utxoSet.Blockchain.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoTableName))

		if b != nil {
			c := b.Cursor()
		UTXOBREAK:
			for k, v := c.First(); k != nil; k, v = c.Next() {
				txOutputs := core.DeserializeTXOutputs(v)
				for _, utxo := range txOutputs.UTXOS {
					if !utxo.Output.UnLockScriptPubKeyWithAddress(from) {
						continue
					}

					money += utxo.Output.Value
					txHash := hex.EncodeToString(utxo.TxHash)
					spentableUTXOs[txHash] = append(spentableUTXOs[txHash], utxo.Index)

					if money >= amount {
						break UTXOBREAK
					}
				}
			}
		}
		return nil
	})

	if money < amount {
		log.Panic("余额不足...")
	}

	return money, spentableUTXOs
}

// Update 更新UTXO集合
func (utxoSet *UTXOSet) Update() {

	block := utxoSet.Blockchain.Iterator().Next()

	ins := []*core.TXInput{}
	outsMap := make(map[string]*core.TXOutputs)

	for _, tx := range block.Txs {
		if tx.IsCoinbaseTransaction() {
			continue
		}
		for _, in := range tx.Vins {
			ins = append(ins, in)
		}
	}

	for _, tx := range block.Txs {
		utxos := []*core.UTXO{}
	WorkOutLoop:
		for index, out := range tx.Vouts {
			for _, in := range ins {
				if in.Vout == index && bytes.Compare(in.TxHash, tx.TxHash) == 0 && bytes.Compare(out.Ripemd160Hash, crypto.Ripemd160Hash(in.PublicKey)) == 0 {
					continue WorkOutLoop
				}
			}

			utxo := &core.UTXO{tx.TxHash, index, out}
			utxos = append(utxos, utxo)
		}
		if len(utxos) > 0 {
			txHash := hex.EncodeToString(tx.TxHash)
			outsMap[txHash] = &core.TXOutputs{utxos}
		}
	}

	err := utxoSet.Blockchain.DB.Update(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(utxoTableName))
		if b != nil {

			// 删除已消费的UTXO
			for _, in := range ins {
				txOutputsBytes := b.Get(in.TxHash)
				if len(txOutputsBytes) == 0 {
					continue
				}
				txOutputs := core.DeserializeTXOutputs(txOutputsBytes)
				UTXOS := []*core.UTXO{}
				isNeedDelete := false

				for _, utxo := range txOutputs.UTXOS {
					if utxo.Index == in.Vout && bytes.Compare(utxo.Output.Ripemd160Hash, crypto.Ripemd160Hash(in.PublicKey)) == 0 {
						isNeedDelete = true
					} else {
						UTXOS = append(UTXOS, utxo)
					}
				}

				if isNeedDelete {
					b.Delete(in.TxHash)
					if len(UTXOS) > 0 {
						txHash := hex.EncodeToString(in.TxHash)
						preTXOutputs := outsMap[txHash]
						if preTXOutputs == nil {
							preTXOutputs = &core.TXOutputs{[]*core.UTXO{}}
						}
						preTXOutputs.UTXOS = append(preTXOutputs.UTXOS, UTXOS...)
						outsMap[txHash] = preTXOutputs
					}
				}
			}

			// 新增UTXO
			for keyHash, outPuts := range outsMap {
				keyHashBytes, _ := hex.DecodeString(keyHash)
				b.Put(keyHashBytes, outPuts.Serialize())
			}
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}
