package network

import (
	"encoding/hex"
	"fmt"
	"learn_DumboNG/011-ACS/core"
	acs "learn_DumboNG/011-ACS/core/acs"
	"learn_DumboNG/011-ACS/store"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type ACSRuntime struct {
	lock       sync.Mutex
	nodeID     uint64
	epoch      uint64
	cfg        acs.Config
	acs        *acs.ACS
	monitor    *acs.DelayMonitor
	blockchain *store.Blockchain
	pool       *TxPool
	proposed   bool
}

func NewACSRuntime(nodeID string, blockchain *store.Blockchain) *ACSRuntime {
	id, err := strconv.ParseUint(nodeID, 10, 64)
	if err != nil {
		panic(err)
	}
	nodes := knownNodeIDs()
	f := (len(nodes) - 1) / 3
	monitor := acs.NewDelayMonitor()
	cfg := acs.Config{N: len(nodes), F: f, ID: id, Nodes: nodes, BatchSize: 100}
	r := &ACSRuntime{nodeID: id, cfg: cfg, monitor: monitor, blockchain: blockchain, pool: NewTxPool()}
	r.resetACS()
	return r
}

func (r *ACSRuntime) AddTransactions(txs []*core.Transaction) {
	r.lock.Lock()
	r.pool.AddBatch(txs)
	shouldPropose := !r.proposed
	r.lock.Unlock()

	if shouldPropose {
		r.Propose()
	}
}

func (r *ACSRuntime) Propose() {
	r.lock.Lock()
	if r.proposed {
		r.lock.Unlock()
		return
	}
	txs := r.pool.Batch(r.cfg.BatchSize)
	if len(txs) == 0 {
		r.lock.Unlock()
		return
	}
	r.proposed = true
	epoch := r.epoch
	proposal := &ACSProposal{Epoch: epoch, ProposerID: r.nodeID, Txs: txs}
	inst := r.acs
	r.lock.Unlock()

	if err := inst.InputValue(encodeProposal(proposal)); err != nil {
		fmt.Printf("ACS input error: %v\n", err)
		return
	}
	r.drainMessages(inst, epoch)
	r.tryCommit()
}

func (r *ACSRuntime) HandleACSMessage(msg *ACSNetMessage) {
	r.lock.Lock()
	if msg.Epoch != r.epoch {
		r.lock.Unlock()
		return
	}
	inst := r.acs
	epoch := r.epoch
	r.lock.Unlock()

	if err := inst.HandleMessage(msg.FromID, msg.Msg); err != nil {
		fmt.Printf("ACS handle error: %v\n", err)
		return
	}
	r.drainMessages(inst, epoch)
	r.tryCommit()
}

func (r *ACSRuntime) drainMessages(inst *acs.ACS, epoch uint64) {
	for _, msg := range inst.Messages() {
		to := addressForNodeID(msg.To)
		if to == "" {
			continue
		}
		acsMsg, ok := msg.Payload.(*acs.ACSMessage)
		if !ok {
			continue
		}
		sendACS(to, &ACSNetMessage{FromID: r.nodeID, Epoch: epoch, Msg: acsMsg})
	}
}

func (r *ACSRuntime) tryCommit() {
	r.lock.Lock()
	inst := r.acs
	r.lock.Unlock()

	output := inst.Output()
	if output == nil {
		return
	}

	txs := mergeACSOutput(output)
	if len(txs) == 0 {
		return
	}

	block := r.blockchain.BuildBlock(txs)
	r.blockchain.CommitBlock(block)
	r.pool.Remove(txs)
	fmt.Printf("ACS epoch %d committed block %x with %d txs\n", r.epoch, block.Hash, len(txs))

	r.lock.Lock()
	r.epoch++
	r.proposed = false
	r.resetACS()
	hasMore := len(r.pool.Batch(r.cfg.BatchSize)) > 0
	r.lock.Unlock()

	if hasMore {
		r.Propose()
	}
}

func (r *ACSRuntime) resetACS() {
	r.cfg.ID = r.nodeID
	r.acs = acs.NewACS(r.cfg, r.monitor)
}

func mergeACSOutput(output map[uint64][]byte) []*core.Transaction {
	ids := make([]uint64, 0, len(output))
	for id := range output {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	seen := make(map[string]bool)
	out := make([]*core.Transaction, 0)
	for _, id := range ids {
		proposal, err := decodeProposal(output[id])
		if err != nil || proposal == nil {
			continue
		}
		for _, tx := range proposal.Txs {
			if tx == nil {
				continue
			}
			key := hex.EncodeToString(tx.TxHash)
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, tx)
		}
	}
	return out
}

func knownNodeIDs() []uint64 {
	nodes := make([]uint64, 0, len(knowNodes))
	for _, addr := range knowNodes {
		parts := strings.Split(addr, ":")
		if len(parts) == 0 {
			continue
		}
		id, err := strconv.ParseUint(parts[len(parts)-1], 10, 64)
		if err == nil {
			nodes = append(nodes, id)
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i] < nodes[j] })
	return nodes
}

func addressForNodeID(id uint64) string {
	needle := fmt.Sprintf(":%d", id)
	for _, addr := range knowNodes {
		if strings.HasSuffix(addr, needle) {
			return addr
		}
	}
	return ""
}
