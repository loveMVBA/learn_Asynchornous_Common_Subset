package network

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"

	"learn_DumboNG/011-ACS/store"
)

func handleVersion(request []byte, bc *store.Blockchain) {

	var buff bytes.Buffer
	var payload Version

	dataBytes := request[COMMANDLENGTH:]

	buff.Write(dataBytes)
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	bestHeight := bc.GetBestHeight()
	foreignerBestHeight := payload.BestHeight
	if bestHeight > foreignerBestHeight {
		sendVersion(payload.AddrFrom, bc)
	} else if bestHeight < foreignerBestHeight {
		sendGetBlocks(payload.AddrFrom)
	}
}

func handleAddr(request []byte, bc *store.Blockchain) {

}

func handleBlock(request []byte, bc *store.Blockchain) {
	var buff bytes.Buffer
	var payload BlockData

	dataBytes := request[COMMANDLENGTH:]

	buff.Write(dataBytes)
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	block := payload.Block

	bc.AddBlock(block)

	if len(transactionArray) == 0 {
		utxoSet := &store.UTXOSet{Blockchain: bc}
		utxoSet.ResetUTXOSet()
	}

	if len(transactionArray) > 0 {
		sendGetData(payload.AddrFrom, BLOCK_TYPE, transactionArray[0])
		transactionArray = transactionArray[1:]
	} else {
		fmt.Println("数据库重置")
		utxoSet := &store.UTXOSet{Blockchain: bc}
		utxoSet.ResetUTXOSet()
	}
}

func handleInv(request []byte, bc *store.Blockchain) {

	var buff bytes.Buffer
	var payload Inv

	dataBytes := request[COMMANDLENGTH:]

	buff.Write(dataBytes)
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	// Inv 3000 block hashes [][]
	if payload.Type == BLOCK_TYPE {
		if len(payload.Items) == 0 {
			return
		}

		blockHash := payload.Items[0]
		sendGetData(payload.AddrFrom, BLOCK_TYPE, blockHash)

		if len(payload.Items) > 1 {
			transactionArray = payload.Items[1:]
		}
	}

	if payload.Type == TX_TYPE {

	}
}

func handleGetBlocks(request []byte, bc *store.Blockchain) {
	var buff bytes.Buffer
	var payload GetBlocks

	dataBytes := request[COMMANDLENGTH:]

	buff.Write(dataBytes)
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	blocks := bc.GetBlockHashes()
	sendInv(payload.AddrFrom, BLOCK_TYPE, blocks)

}

func handleGetData(request []byte, bc *store.Blockchain) {
	var buff bytes.Buffer
	var payload GetData

	dataBytes := request[COMMANDLENGTH:]

	buff.Write(dataBytes)
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	if payload.Type == BLOCK_TYPE {
		block, err := bc.GetBlock(payload.Hash)
		if err != nil || block == nil {
			return
		}
		sendBlock(payload.AddrFrom, block)
	}

	if payload.Type == TX_TYPE {

	}
}

func handleTx(request []byte, bc *store.Blockchain) {
	var payload TxMessage
	if err := decodeGobPayload(request, &payload); err != nil {
		log.Panic(err)
	}
	if consensusRuntime == nil {
		return
	}
	consensusRuntime.AddTransactions(payload.Txs)
}

func handleACS(request []byte, bc *store.Blockchain) {
	var payload ACSNetMessage
	if err := decodeGobPayload(request, &payload); err != nil {
		log.Panic(err)
	}
	if consensusRuntime == nil {
		return
	}
	consensusRuntime.HandleACSMessage(&payload)
}
