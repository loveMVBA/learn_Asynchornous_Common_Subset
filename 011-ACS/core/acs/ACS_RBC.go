package ACS

/* RBC (可靠广播, Reliable Broadcast)
可靠广播是一个通信原语，用于确保网络中的一个节点能够将消息可靠、一致地发送给所有其他节点，即使部分节点会发生故障或恶意攻击。

Proposer 输入原始数据
        |
        v
切片 + Reed-Solomon 编码 + Merkle proof
        |
        v
ProofRequest 初始发送
        |
        v
EchoRequest 广播
        |
        v
ReadyRequest 广播
        |
        v
收集足够 Echo + Ready 后重建原始数据

*/

import (
	"bytes"
	"fmt"
	"sort"
	"sync"

	"github.com/klauspost/reedsolomon"
)

type RBC struct {
	Config
	proposerID uint64
	recvReadys map[uint64]*ReadyRequest
	recvEchos  map[uint64]*EchoRequest

	encoder         reedsolomon.Encoder
	numParityShards int
	numDataShards   int

	echoSent  bool
	readySent bool
	decided   bool

	lock     sync.Mutex
	messages []*BroadcastMessage

	output []byte

	closeCh   chan struct{}
	inputCh   chan rbcInput
	messageCh chan rbcMessage

	delayMonitor *DelayMonitor
}

type rbcMessage struct {
	senderID uint64
	msg      *BroadcastMessage
	err      chan error
}

type rbcInputResponse struct {
	messages []*BroadcastMessage
	err      error
}

type rbcInput struct {
	value    []byte
	response chan rbcInputResponse
}

type ProofRequest struct {
	RootHash []byte
	Proof    [][]byte
	Index    int
	Leaves   int
}

type EchoRequest struct {
	ProofRequest
}

type ReadyRequest struct {
	ProofRequest
}

type proofList []ProofRequest

func NewRBC(cfg Config, proposerID uint64, delayMonitor *DelayMonitor) *RBC {
	parityShards := 2 * cfg.F
	dataShards := cfg.N - parityShards

	encoder, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		panic(err)
	}

	rbc := &RBC{
		Config:          cfg,
		proposerID:      proposerID,
		recvReadys:      make(map[uint64]*ReadyRequest),
		recvEchos:       make(map[uint64]*EchoRequest),
		encoder:         encoder,
		numParityShards: parityShards,
		numDataShards:   dataShards,
		messages:        make([]*BroadcastMessage, 0),
		closeCh:         make(chan struct{}),
		inputCh:         make(chan rbcInput),
		messageCh:       make(chan rbcMessage),
		delayMonitor:    delayMonitor,
	}

	go rbc.run()
	return rbc
}

// 输入，然后返回结果
func (r *RBC) InputValue(data []byte) ([]*BroadcastMessage, error) {
	input := rbcInput{
		value:    data,
		response: make(chan rbcInputResponse),
	}
	r.inputCh <- input
	resp := <-input.response
	return resp.messages, resp.err
}

// 输入，然后返回错误
func (r *RBC) HandleMessage(senderID uint64, msg *BroadcastMessage) error {
	m := rbcMessage{
		senderID: senderID,
		msg:      msg,
		err:      make(chan error),
	}
	r.messageCh <- m
	return <-m.err
}

// 返回messages，并清空
func (r *RBC) Messages() []*BroadcastMessage {
	r.lock.Lock()
	defer r.lock.Unlock()

	msgs := r.messages
	r.messages = []*BroadcastMessage{}
	return msgs
}

// 返回output，并清空
func (r *RBC) Output() []byte {
	out := r.output
	r.output = nil
	return out
}

func (r *RBC) run() {
	for {
		select {
		case t := <-r.inputCh:
			msgs, err := r.inputValue(t.value)
			t.response <- rbcInputResponse{messages: msgs, err: err}
		case t := <-r.messageCh:
			t.err <- r.handleMessage(t.senderID, t.msg)
		case <-r.closeCh:
			return
		}
	}
}

func (r *RBC) inputValue(data []byte) ([]*BroadcastMessage, error) {
	// 开始RBC延迟计时
	r.delayMonitor.StartRBC(0) // 使用0作为开始的epoch
	//里德所罗门编码，可以将消息拆成多个部分，然后用这些部分的叫小部分集合进行重构
	shards, err := r.encoder.Split(data)
	if err != nil {
		return nil, err
	}
	if err := r.encoder.Encode(shards); err != nil {
		return nil, err
	}
	msgs := make([]*BroadcastMessage, len(shards))
	// 创建默克尔树
	mt := NewMerkleTree(shards)
	root := mt.Root()

	for i := 0; i < len(msgs); i++ {
		proof, err := mt.GetProof(i)
		if err != nil {
			return nil, err
		}

		msgs[i] = &BroadcastMessage{
			Payload: &ProofRequest{
				RootHash: root,
				Proof:    proof,
				Index:    i,
				Leaves:   len(shards),
			},
		}
	}

	ownIndex := 0
	for i, id := range r.Nodes {
		if id == r.ID {
			ownIndex = i
			break
		}
	}

	proof, ok := msgs[ownIndex].Payload.(*ProofRequest)
	if !ok {
		return nil, fmt.Errorf("payload decode err")
	}

	if err := r.handleProofRequest(r.ID, proof); err != nil {
		return nil, err
	}

	out := make([]*BroadcastMessage, 0, len(msgs)-1)
	for i, msg := range msgs {
		if i != ownIndex {
			out = append(out, msg)
		}
	}

	return out, nil
}

// 根据msg类型分别处理
func (r *RBC) handleMessage(senderID uint64, msg *BroadcastMessage) error {
	switch t := msg.Payload.(type) {
	case *ProofRequest:
		return r.handleProofRequest(senderID, t)
	case *EchoRequest:
		return r.handleEchoRequest(senderID, t)
	case *ReadyRequest:
		return r.handleReadyRequest(senderID, t)
	}
	return nil
}

/*
处理来自proposer的proof，验证合法之后广播这个proof
节点收到合法 ProofRequest

	|
	v

广播 EchoRequest

	|
	v

本地也记录自己的 Echo
*/
func (r *RBC) handleProofRequest(senderID uint64, req *ProofRequest) error {
	if senderID != r.proposerID {
		return fmt.Errorf("senderId != proposerId")
	}
	if r.echoSent {
		return fmt.Errorf("proof has recvieved")
	}
	if len(req.Proof) == 0 || !VerifyMerkleProofSimple(req.Proof[0], req.Proof, req.RootHash, req.Index) {
		return fmt.Errorf("invalid proof")
	}

	r.echoSent = true
	echo := &EchoRequest{*req}
	// 广播收到的proof
	r.messages = append(r.messages, &BroadcastMessage{echo})
	return r.handleEchoRequest(r.ID, echo)
}

/*
	处理echo，判断是否ready

收到 Echo

	|
	v

记录 Echo

	|
	v

如果相同 root 的 Echo 数量 >= N-F

	|
	v

广播 Ready
*/
func (r *RBC) handleEchoRequest(senderID uint64, req *EchoRequest) error {
	if len(req.Proof) == 0 || !VerifyMerkleProofSimple(req.Proof[0], req.Proof, req.RootHash, req.Index) {
		return fmt.Errorf("invalid proof")
	}

	r.recvEchos[senderID] = req
	if r.countEchos(req.RootHash) < r.N-r.F {
		return nil
	}
	if r.readySent {
		return r.decodeValue(req.RootHash)
	}

	r.readySent = true
	ready := &ReadyRequest{ProofRequest{RootHash: req.RootHash}}
	r.messages = append(r.messages, &BroadcastMessage{ready})
	return r.handleReadyRequest(r.ID, ready)
}

// 处理ready，尝试decode
func (r *RBC) handleReadyRequest(senderID uint64, req *ReadyRequest) error {
	if _, ok := r.recvReadys[senderID]; ok {
		return fmt.Errorf("already received ready from server %d", senderID)
	}
	r.recvReadys[senderID] = req

	//如果我还没发过 Ready，但是我已经看到 F+1 个节点发 Ready，
	//那我也跟着发 Ready。
	if r.countReadys(req.RootHash) == r.F+1 && !r.readySent {
		r.readySent = true
		ready := &ReadyRequest{ProofRequest{RootHash: req.RootHash}}
		r.messages = append(r.messages, &BroadcastMessage{ready})
	}
	return r.decodeValue(req.RootHash)
}

func (r *RBC) decodeValue(hash []byte) error {
	// 通过条件需要countReadys(hash) > 2F
	// 并且 countEchos(hash) > F
	if r.decided || r.countReadys(hash) <= 2*r.F || r.countEchos(hash) <= r.F {
		return nil
	}

	r.decided = true
	var proof proofList
	for _, echo := range r.recvEchos {
		proof = append(proof, echo.ProofRequest)
	}
	sort.Sort(proof)

	// 重建shards
	shards := make([][]byte, r.numParityShards+r.numDataShards)
	for _, p := range proof {
		shards[p.Index] = p.Proof[0]
	}
	if err := r.encoder.Reconstruct(shards); err != nil {
		return nil
	}

	var result []byte
	for _, data := range shards[:r.numDataShards] {
		result = append(result, data...)
	}
	r.output = result

	// 结束RBC延迟计时并记录
	delay := r.delayMonitor.EndRBC(0) // 使用0作为epoch
	fmt.Printf("RBC completed with delay: %v\n", delay)

	return nil
}

// utils
func (r *RBC) countEchos(hash []byte) int {
	n := 0
	for _, h := range r.recvEchos {
		if bytes.Compare(hash, h.RootHash) == 0 {
			n++
		}
	}
	return n
}

func (r *RBC) countReadys(hash []byte) int {
	n := 0
	for _, r := range r.recvReadys {
		if bytes.Compare(hash, r.RootHash) == 0 {
			n++
		}
	}
	return n
}

func (p proofList) Len() int {
	return len(p)
}

func (p proofList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p proofList) Less(i, j int) bool {
	return p[i].Index < p[j].Index
}
