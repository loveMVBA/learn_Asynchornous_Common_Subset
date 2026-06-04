package node

import (
	"flag"
	"fmt"
	"learn_DumboNG/011-ACS/crypto"
	"log"
	"os"
)

type CLI struct{}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("\taddresslists -- 输出当前节点所有钱包地址列表")
	fmt.Println("\tcreatewallet -- 创建当前节点新钱包")
	fmt.Println("\tcreateblockchain -address ADDRESS -- 创建当前节点创世区块")
	fmt.Println("\tsend -from FROM -to TO -amount AMOUNT [-mine] -- 交易明细")
	fmt.Println("\tprintchain -- 输出当前节点区块信息")
	fmt.Println("\tgetbalance -address ADDRESS -- 查询当前节点余额")
	fmt.Println("\tresetUTXO -- 重置当前节点 UTXO")
	fmt.Println("\tstartnode [-miner ADDRESS] -- 启动当前节点")
}

func isValidArgs() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
}

func (cli *CLI) Run() {
	isValidArgs()

	// 获取节点ID，例如 3000、3001、3002。
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		fmt.Printf("NODE_ID not set\n")
		os.Exit(1)
	}

	addressListsCmd := flag.NewFlagSet("addresslists", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	sendBlockCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	getbalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	resetUTXOCmd := flag.NewFlagSet("resetUTXO", flag.ExitOnError)
	startNodeCmd := flag.NewFlagSet("startnode", flag.ExitOnError)

	flagFrom := sendBlockCmd.String("from", "", "转账来源地址")
	flagTo := sendBlockCmd.String("to", "", "转账目的地地址")
	flagAmount := sendBlockCmd.String("amount", "", "转账金额")
	flagMine := sendBlockCmd.Bool("mine", false, "是否在当前节点验证")
	flagMiner := startNodeCmd.String("miner", "", "定义挖矿奖励的地址")

	flagCreateBlockchainWithAddress := createBlockchainCmd.String("address", "", "创世区块奖励地址")
	getbalanceWithAddress := getbalanceCmd.String("address", "", "查询余额地址")

	switch os.Args[1] {
	case "send":
		err := sendBlockCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "getbalance":
		err := getbalanceCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "addresslists":
		err := addressListsCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "resetUTXO":
		err := resetUTXOCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "startnode":
		err := startNodeCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		printUsage()
		os.Exit(1)
	}

	if sendBlockCmd.Parsed() {
		if *flagFrom == "" || *flagTo == "" || *flagAmount == "" {
			printUsage()
			os.Exit(1)
		}

		from := crypto.JSONToArray(*flagFrom)
		to := crypto.JSONToArray(*flagTo)
		for index, fromAddress := range from {
			if crypto.IsValidForAddress([]byte(fromAddress)) == false || crypto.IsValidForAddress([]byte(to[index])) == false {
				fmt.Println("地址无效.....")
				printUsage()
				os.Exit(1)
			}
		}
		amount := crypto.JSONToArray(*flagAmount)
		cli.send(from, to, amount, nodeID, *flagMine)
	}

	if printChainCmd.Parsed() {
		cli.printchain(nodeID)
	}

	if createWalletCmd.Parsed() {
		cli.CreateWallet(nodeID)
	}

	if addressListsCmd.Parsed() {
		cli.addressLists(nodeID)
	}

	if createBlockchainCmd.Parsed() {
		if *flagCreateBlockchainWithAddress == "" || crypto.IsValidForAddress([]byte(*flagCreateBlockchainWithAddress)) == false {
			fmt.Println("地址无效...")
			printUsage()
			os.Exit(1)
		}
		cli.createGenesisBlockchain(*flagCreateBlockchainWithAddress, nodeID)
	}

	if getbalanceCmd.Parsed() {
		if crypto.IsValidForAddress([]byte(*getbalanceWithAddress)) == false {
			fmt.Println("地址无效...")
			printUsage()
			os.Exit(1)
		}
		cli.getBalance(*getbalanceWithAddress, nodeID)
	}

	if resetUTXOCmd.Parsed() {
		cli.resetUTXOSet(nodeID)
	}

	if startNodeCmd.Parsed() {
		cli.startNode(nodeID, *flagMiner)
	}
}
