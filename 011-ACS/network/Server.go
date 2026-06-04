package network

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"learn_DumboNG/011-ACS/store"
)

func StartServer(nodeID string, minerAdd string) {
	registerConsensusMessages()
	nodeAddress = fmt.Sprintf("localhost:%s", nodeID)

	if store.DBExists(nodeID) == false {
		store.InitEmptyBlockchainDB(nodeID)
	}

	ln, err := net.Listen(PROTOCOL, nodeAddress)
	if err != nil {
		log.Panic(err)
	}
	defer ln.Close()

	bc := store.BlockchainObject(nodeID)
	consensusRuntime = NewACSRuntime(nodeID, bc)

	// 第一个终端：端口3000，启动主节点
	// 第二个终端：端口3001，钱包节点
	// 第三个终端：端口3002，矿工节点
	if nodeAddress != knowNodes[0] {
		// 此节点是钱包或矿工，需要向主节点发送请求同步数据
		sendVersion(knowNodes[0], bc)
	}

	for {
		// 接受客户端发送的数据
		// 收到数据的格式是固定：COMMANDLENGTH字节的命令 + 结构体的字节数组
		conn, err := ln.Accept()
		if err != nil {
			log.Panic(err)
		}
		go handleConnection(conn, bc)
	}
}

func handleConnection(conn net.Conn, bc *store.Blockchain) {
	defer conn.Close()

	request, err := ioutil.ReadAll(conn)
	if err != nil {
		log.Panic(err)
	}
	if len(request) < COMMANDLENGTH {
		fmt.Printf("[%s] <- [%s] Receive invalid message\n", nodeAddress, conn.RemoteAddr())
		return
	}

	command := bytesToCommand(request[:COMMANDLENGTH])
	from := messageAddrFrom(command, request)
	if from == "" {
		from = conn.RemoteAddr().String()
	}
	fmt.Printf("[%s] <- [%s] Receive Message: %s\n", nodeAddress, from, command)
	switch command {
	case COMMAND_VERSION:
		handleVersion(request, bc)
	case COMMAND_ADDR:
		handleAddr(request, bc)
	case COMMAND_BLOCK:
		handleBlock(request, bc)
	case COMMAND_INV:
		handleInv(request, bc)
	case COMMAND_GET_BLOCKS:
		handleGetBlocks(request, bc)
	case COMMAND_GETDATA:
		handleGetData(request, bc)
	case COMMAND_TX:
		handleTx(request, bc)
	case COMMAND_ACS:
		handleACS(request, bc)
	default:
		fmt.Printf("[%s] Unknown command: %s\n", nodeAddress, command)
	}
}

func messageAddrFrom(command string, request []byte) string {
	if len(request) <= COMMANDLENGTH {
		return ""
	}

	var buff bytes.Buffer
	buff.Write(request[COMMANDLENGTH:])
	dec := gob.NewDecoder(&buff)

	switch command {
	case COMMAND_VERSION:
		var payload Version
		if err := dec.Decode(&payload); err != nil {
			return ""
		}
		return payload.AddrFrom
	case COMMAND_BLOCK:
		var payload BlockData
		if err := dec.Decode(&payload); err != nil {
			return ""
		}
		return payload.AddrFrom
	case COMMAND_INV:
		var payload Inv
		if err := dec.Decode(&payload); err != nil {
			return ""
		}
		return payload.AddrFrom
	case COMMAND_GET_BLOCKS:
		var payload GetBlocks
		if err := dec.Decode(&payload); err != nil {
			return ""
		}
		return payload.AddrFrom
	case COMMAND_GETDATA:
		var payload GetData
		if err := dec.Decode(&payload); err != nil {
			return ""
		}
		return payload.AddrFrom
	case COMMAND_ACS:
		var payload ACSNetMessage
		if err := dec.Decode(&payload); err != nil {
			return ""
		}
		return addressForNodeID(payload.FromID)
	default:
		return ""
	}
}
