package core

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"learn_DumboNG/011-ACS/crypto"
	"log"
	"math/big"
	"time"
)

// UTXOProvider 提供创建普通交易所需的 UTXO 查询和签名能力。
type UTXOProvider interface {
	FindSpendableUTXOs(from string, amount int64, txs []*Transaction) (int64, map[string][]int)
	SignTransaction(tx *Transaction, privateKey ecdsa.PrivateKey, txs []*Transaction)
}

// UTXO交易模型
type Transaction struct {
	//1.交易hash ()
	TxHash []byte

	//2.输入
	Vins []*TXInput

	//3.输出
	Vouts []*TXOutput
}

// 判断当前交易是否是Coinbase交易
func (tx *Transaction) IsCoinbaseTransaction() bool {
	return len(tx.Vins) == 1 && len(tx.Vins[0].TxHash) == 0 && tx.Vins[0].Vout == -1
}

// transaction创建分两种情况
// 创世区块创建时的transaction
func NewCoinbaseTransaction(address string) *Transaction {

	txInput := &TXInput{[]byte{}, -1, nil, []byte{}}

	txOutput := NewTXOutput(10, address)

	txCoinbase := &Transaction{[]byte{}, []*TXInput{txInput}, []*TXOutput{txOutput}}

	//设置hash值
	txCoinbase.HashTransaction()

	return txCoinbase
}

func (tx *Transaction) HashTransaction() {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	resultBytes := bytes.Join([][]byte{crypto.IntTOHex(time.Now().Unix()), result.Bytes()}, []byte{})

	hash := sha256.Sum256(resultBytes)
	tx.TxHash = hash[:]
}

// 转账时产生的transaction
func NewSimpleTransaction(from string, to string, amount int64, utxoSet UTXOProvider, txs []*Transaction, nodeID string) *Transaction {

	wallets, _ := crypto.NewWallets(nodeID)
	wallet := wallets.WalletsMap[from]
	// 有一个函数，返回一个数组里面是from这个人所有未花费输出所对应的Transaction
	money, spendableUTXODic := utxoSet.FindSpendableUTXOs(from, amount, txs)

	var txInputs []*TXInput
	var txOutputs []*TXOutput
	for txHash, indexArray := range spendableUTXODic {
		txHashBytes, _ := hex.DecodeString(string(txHash))
		//消费
		for _, index := range indexArray {
			txInput := &TXInput{txHashBytes, index, nil, wallet.PublicKey}
			txInputs = append(txInputs, txInput)
		}
	}

	//转账
	txOutput := NewTXOutput(int64(amount), to)
	txOutputs = append(txOutputs, txOutput)

	//找零
	txOutput = NewTXOutput(int64(money)-int64(amount), from)
	txOutputs = append(txOutputs, txOutput)

	//设置hash
	tx := &Transaction{[]byte{}, txInputs, txOutputs}

	//进行签名
	utxoSet.SignTransaction(tx, wallet.PrivateKey, txs)
	tx.TxHash = tx.Hash()

	return tx
}

func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTXs map[string]Transaction) {
	if tx.IsCoinbaseTransaction() {
		return
	}

	for _, vin := range tx.Vins {
		if prevTXs[hex.EncodeToString(vin.TxHash)].TxHash == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy()

	for inID, vin := range txCopy.Vins {
		prevTx := prevTXs[hex.EncodeToString(vin.TxHash)]
		txCopy.Vins[inID].Signature = nil
		txCopy.Vins[inID].PublicKey = prevTx.Vouts[vin.Vout].Ripemd160Hash
		txCopy.TxHash = txCopy.Hash()
		txCopy.Vins[inID].PublicKey = nil

		r, s, err := ecdsa.Sign(rand.Reader, &privKey, txCopy.TxHash)
		if err != nil {
			log.Panic(err)
		}
		signature := append(r.Bytes(), s.Bytes()...)

		tx.Vins[inID].Signature = signature
	}
}

// 拷贝Tx用于签名
func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []*TXInput
	var outputs []*TXOutput

	for _, vin := range tx.Vins {
		inputs = append(inputs, &TXInput{vin.TxHash, vin.Vout, nil, nil})
	}

	for _, vout := range tx.Vouts {
		outputs = append(outputs, &TXOutput{vout.Value, vout.Ripemd160Hash})
	}

	txCopy := Transaction{tx.TxHash, inputs, outputs}
	return txCopy
}

// 计算交易hash
func (tx *Transaction) Hash() []byte {
	txCopy := tx
	txCopy.TxHash = []byte{}
	hash := sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

func (tx *Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}
	return encoded.Bytes()
}

// 验证交易签名
func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	if tx.IsCoinbaseTransaction() {
		return true
	}

	for _, vin := range tx.Vins {
		if prevTXs[hex.EncodeToString(vin.TxHash)].TxHash == nil {
			log.Panic("ERROR: 上一次交易不正确")
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()
	for inID, vin := range tx.Vins {
		prevTx := prevTXs[hex.EncodeToString(vin.TxHash)]
		txCopy.Vins[inID].Signature = nil
		txCopy.Vins[inID].PublicKey = prevTx.Vouts[vin.Vout].Ripemd160Hash
		txCopy.TxHash = txCopy.Hash()
		txCopy.Vins[inID].PublicKey = nil

		r := big.Int{}
		s := big.Int{}
		sigLen := len(vin.Signature)
		r.SetBytes(vin.Signature[:(sigLen / 2)])
		s.SetBytes(vin.Signature[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}
		keyLen := len(vin.PublicKey)
		x.SetBytes(vin.PublicKey[:(keyLen / 2)])
		y.SetBytes(vin.PublicKey[(keyLen / 2):])
		rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}

		if ecdsa.Verify(&rawPubKey, txCopy.TxHash, &r, &s) == false {
			return false
		}
	}

	return true
}
