package core

import (
	"bytes"
	"learn_DumboNG/011-ACS/crypto"
)

type TXOutput struct {
	Value         int64
	Ripemd160Hash []byte //用户名
}

func (txOutput *TXOutput) Lock(address string) {
	publicKeyHash := crypto.Base58Decode([]byte(address))
	txOutput.Ripemd160Hash = publicKeyHash[1 : len(publicKeyHash)-4]

}

func NewTXOutput(value int64, address string) *TXOutput {
	txOutput := &TXOutput{value, nil}

	// 设置Ripemd160Hash
	txOutput.Lock(address)

	return txOutput
}

func (txOutput *TXOutput) UnLockScriptPubKeyWithAddress(address string) bool {

	publicKeyHash := crypto.Base58Decode([]byte(address))
	hash160 := publicKeyHash[1 : len(publicKeyHash)-4]

	return bytes.Equal(txOutput.Ripemd160Hash, hash160)
}
