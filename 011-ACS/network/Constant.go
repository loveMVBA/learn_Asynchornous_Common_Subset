package network

const PROTOCOL = "tcp"
const COMMANDLENGTH = 12
const NODE_VERSION = 1

//发送数据的时候是 （12个字节 + 结构体序列化的字节数组

// 命令
const COMMAND_VERSION = "version"
const COMMAND_ADDR = "addr"
const COMMAND_BLOCK = "block"
const COMMAND_INV = "inv"
const COMMAND_GET_BLOCKS = "getblocks"
const COMMAND_GETDATA = "getdata"
const COMMAND_TX = "tx"
const COMMAND_ACS = "acs"

// 类型
const BLOCK_TYPE = "block"
const TX_TYPE = "tx"
