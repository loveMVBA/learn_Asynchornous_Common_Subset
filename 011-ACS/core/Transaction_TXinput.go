package core

import (
	"bytes"
	"learn_DumboNG/011-ACS/crypto"
)

type TXInput struct {
	//1.交易hash
	TxHash []byte
	//2.存储TXoutput的Vout的索引
	Vout int
	//3.数字签名
	Signature []byte
	//4.公钥
	PublicKey []byte
}

// 判断当前的消费是属于谁的钱
func (txInput *TXInput) UnLockRipemd160(ripmd160Hash []byte) bool {

	publicKey := crypto.Ripemd160Hash(txInput.PublicKey)

	return bytes.Equal(publicKey, ripmd160Hash)
}
