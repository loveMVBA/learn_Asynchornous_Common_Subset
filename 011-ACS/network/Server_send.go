package network

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"

	"learn_DumboNG/011-ACS/core"
	"learn_DumboNG/011-ACS/store"
)

func sendMessage() {
	fmt.Println("客户端向服务器发送数据...")
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		panic("error")
	}
	defer conn.Close()

	_, err = io.Copy(conn, bytes.NewReader([]byte("version")))
	if err != nil {
		log.Panic(err)
	}

}

func sendVersion(toAddress string, bc *store.Blockchain) {

	bestHeight := bc.GetBestHeight()
	payload := gobEncode(Version{NODE_VERSION, bestHeight, nodeAddress})

	request := append(commandToBytes(COMMAND_VERSION), payload...)
	sendData(toAddress, request)
}

func sendData(to string, data []byte) {
	command := "unknown"
	if len(data) >= COMMANDLENGTH {
		command = bytesToCommand(data[:COMMANDLENGTH])
	}
	from := nodeAddress
	if from == "" {
		from = "unknown"
	}
	fmt.Printf("[%s] -> [%s] Send Message: %s\n", from, to, command)

	conn, err := net.Dial("tcp", to)
	if err != nil {
		fmt.Printf("[%s] -> [%s] Send failed: %v\n", from, to, err)
		return
	}
	defer conn.Close()

	_, err = io.Copy(conn, bytes.NewReader(data))
	if err != nil {
		log.Panic(err)
	}
}

func sendGetBlocks(toAddress string) {
	payload := gobEncode(GetBlocks{nodeAddress})
	request := append(commandToBytes(COMMAND_GET_BLOCKS), payload...)
	sendData(toAddress, request)
}

// 主节点将自己所有的区块hash发送到钱包节点
func sendInv(toAddress string, command string, hashes [][]byte) {
	payload := gobEncode(Inv{nodeAddress, command, hashes})
	request := append(commandToBytes(COMMAND_INV), payload...)
	sendData(toAddress, request)
}

func sendGetData(toAddress string, kind string, blockhash []byte) {
	payload := gobEncode(GetData{nodeAddress, kind, blockhash})
	request := append(commandToBytes(COMMAND_GETDATA), payload...)
	sendData(toAddress, request)
}

func sendBlock(toAddress string, block *core.Block) {
	payload := gobEncode(BlockData{nodeAddress, block})
	request := append(commandToBytes(COMMAND_BLOCK), payload...)
	sendData(toAddress, request)
}

func SubmitTxs(fromID string, txs []*core.Transaction) {
	id, err := strconv.ParseUint(fromID, 10, 64)
	if err != nil {
		log.Panic(err)
	}
	payload := gobEncode(TxMessage{FromID: id, Txs: txs})
	request := append(commandToBytes(COMMAND_TX), payload...)
	for _, node := range knowNodes {
		sendData(node, request)
	}
}

func sendACS(toAddress string, msg *ACSNetMessage) {
	payload := gobEncode(msg)
	request := append(commandToBytes(COMMAND_ACS), payload...)
	sendData(toAddress, request)
}
