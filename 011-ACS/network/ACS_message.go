package network

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"learn_DumboNG/011-ACS/core"
	acs "learn_DumboNG/011-ACS/core/acs"
)

type TxMessage struct {
	FromID uint64
	Txs    []*core.Transaction
}

type ACSNetMessage struct {
	FromID uint64
	Epoch  uint64
	Msg    *acs.ACSMessage
}

type ACSProposal struct {
	Epoch      uint64
	ProposerID uint64
	Txs        []*core.Transaction
}

func encodeProposal(p *ACSProposal) []byte {
	var buff bytes.Buffer
	if err := gob.NewEncoder(&buff).Encode(p); err != nil {
		panic(err)
	}
	return buff.Bytes()
}

func decodeProposal(data []byte) (*ACSProposal, error) {
	var p ACSProposal
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

func registerConsensusMessages() {
	gob.Register(&TxMessage{})
	gob.Register(&ACSNetMessage{})
	gob.Register(&ACSProposal{})
	gob.Register(&acs.ACSMessage{})
	gob.Register(&acs.BroadcastMessage{})
	gob.Register(&acs.AgreementMessage{})
	gob.Register(&acs.ProofRequest{})
	gob.Register(&acs.EchoRequest{})
	gob.Register(&acs.ReadyRequest{})
	gob.Register(&acs.BinaryValueRequest{})
	gob.Register(&acs.AuxRequest{})
}

func decodeGobPayload(data []byte, out interface{}) error {
	if len(data) < COMMANDLENGTH {
		return fmt.Errorf("message too short")
	}
	return gob.NewDecoder(bytes.NewReader(data[COMMANDLENGTH:])).Decode(out)
}
