package crypto

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

const walletFile = "wallets.dat"

type Wallets struct {
	WalletsMap map[string]*Wallet
}

// SaveWallets 保存钱包数据
func (w *Wallets) SaveWallets(nodeID string) {

	walletFile := fmt.Sprintf("%s_%s", nodeID, walletFile)

	var content bytes.Buffer

	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(&w)
	if err != nil {
		log.Panic(err)
	}

	// 保存序列化后钱包数据到文件
	err = ioutil.WriteFile(walletFile, content.Bytes(), 0644)
	if err != nil {
		log.Panic(err)
	}
}

func NewWallets(nodeID string) (*Wallets, error) {

	walletFile := fmt.Sprintf("%s_%s", nodeID, walletFile)

	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		wallets := &Wallets{}
		wallets.WalletsMap = make(map[string]*Wallet)
		return wallets, err
	}

	filecontent, err := ioutil.ReadFile(walletFile)
	if err != nil {
		log.Panic(err)
	}

	var wallets Wallets
	decoder := gob.NewDecoder(bytes.NewReader(filecontent))
	err = decoder.Decode(&wallets)
	if err != nil {
		log.Panic(err)
	}
	return &wallets, nil
}

func (w *Wallets) CreateNewWallet(nodeID string) {
	wallet := NewWallet()
	fmt.Printf("Address: %s\n", wallet.GetAddress())
	w.WalletsMap[string(wallet.GetAddress())] = wallet

	w.SaveWallets(nodeID)
}
