package network

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
)

func gobEncode(data interface{}) []byte {
	var buff bytes.Buffer
	encoder := gob.NewEncoder(&buff)
	if err := encoder.Encode(data); err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}

func commandToBytes(command string) []byte {
	var bytes [COMMANDLENGTH]byte

	for i, c := range command {
		bytes[i] = byte(c)
	}
	return bytes[:]
}

func bytesToCommand(bytes []byte) string {
	var command []byte

	for _, b := range bytes {
		if b != 0x0 {
			command = append(command, b)
		}
	}
	return fmt.Sprintf("%s", command)
}
