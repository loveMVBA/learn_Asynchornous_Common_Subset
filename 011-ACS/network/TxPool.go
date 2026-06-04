package network

import (
	"encoding/hex"
	"learn_DumboNG/011-ACS/core"
	"sort"
	"sync"
)

type TxPool struct {
	lock sync.Mutex
	txs  map[string]*core.Transaction
}

func NewTxPool() *TxPool {
	return &TxPool{txs: make(map[string]*core.Transaction)}
}

func (p *TxPool) AddBatch(txs []*core.Transaction) {
	p.lock.Lock()
	defer p.lock.Unlock()
	for _, tx := range txs {
		if tx == nil || len(tx.TxHash) == 0 {
			continue
		}
		p.txs[hex.EncodeToString(tx.TxHash)] = tx
	}
}

func (p *TxPool) Batch(limit int) []*core.Transaction {
	p.lock.Lock()
	defer p.lock.Unlock()
	keys := make([]string, 0, len(p.txs))
	for key := range p.txs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if limit <= 0 || limit > len(keys) {
		limit = len(keys)
	}
	out := make([]*core.Transaction, 0, limit)
	for _, key := range keys[:limit] {
		out = append(out, p.txs[key])
	}
	return out
}

func (p *TxPool) Remove(txs []*core.Transaction) {
	p.lock.Lock()
	defer p.lock.Unlock()
	for _, tx := range txs {
		if tx != nil {
			delete(p.txs, hex.EncodeToString(tx.TxHash))
		}
	}
}
