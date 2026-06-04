package ACS

/*
Common Coin （公共硬币）
让所有诚实节点在同一轮最终得到同一个随机布尔值，用于推动 BBA 终止。
这里使用的是TBLS阈值签名来实现
*/

import (
	"crypto/sha256"
	"fmt"
	"sync"
)

// 公共硬币实现
type CommonCoin struct {
	Config
	// 阈值签名公钥
	PK *TBLSPublicKey
	// 阈值签名私钥
	SK *TBLSPrivateKey
	// 接收到的签名份额
	receivedShares map[uint64]map[uint64]*Point // round -> sender -> signature
	// 输出队列
	outputQueues map[uint64]chan bool
	// 锁
	lock sync.RWMutex
	// 消息通道
	messageCh chan coinMessage
	// 关闭通道
	closeCh chan struct{}
}

type coinMessage struct {
	senderID uint64
	round    uint64
	sig      *Point
}

// 创建公共硬币实例
func NewCommonCoin(cfg Config, pk *TBLSPublicKey, sk *TBLSPrivateKey) *CommonCoin {
	cc := &CommonCoin{
		Config:         cfg,
		PK:             pk,
		SK:             sk,
		receivedShares: make(map[uint64]map[uint64]*Point),
		outputQueues:   make(map[uint64]chan bool),
		messageCh:      make(chan coinMessage, 1024),
		closeCh:        make(chan struct{}),
	}

	go cc.run()
	return cc
}

// 运行公共硬币
func (cc *CommonCoin) run() {
	for {
		select {
		case msg := <-cc.messageCh:
			cc.handleMessage(msg.senderID, msg.round, msg.sig)
		case <-cc.closeCh:
			return
		}
	}
}

/*
	处理消息

收到某个节点对某一轮的签名份额

	|
	v

检查该 round 的 map 是否初始化

	|
	v

检查是否重复收到同一个 sender 的份额

	|
	v

验证签名份额

	|
	v

保存签名份额

	|
	v

如果收到 >= F+1 个份额

	|
	v

组合签名

	|
	v

验证组合签名

	|
	v

对组合签名 hash，得到 bool 硬币值

	|
	v

写入 outputQueues[round]
*/
func (cc *CommonCoin) handleMessage(senderID, round uint64, sig *Point) {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	// 初始化轮次
	if cc.receivedShares[round] == nil {
		cc.receivedShares[round] = make(map[uint64]*Point)
	}

	// 检查是否已经收到过该发送者的签名
	if _, exists := cc.receivedShares[round][senderID]; exists {
		return // 重复消息，忽略
	}

	// 验证签名
	message := fmt.Sprintf("coin_%d_%d", cc.ID, round)
	messageBytes := []byte(message)

	if !cc.PK.VerifyShare(sig, int(senderID), messageBytes) {
		return // 签名验证失败
	}

	// 存储签名份额
	cc.receivedShares[round][senderID] = sig

	// 检查是否达到阈值
	if len(cc.receivedShares[round]) >= cc.F+1 {
		// 组合签名份额
		sigs := make(map[int]*Point)
		count := 0
		for sender, sig := range cc.receivedShares[round] {
			if count >= cc.F+1 {
				break
			}
			sigs[int(sender)] = sig
			count++
		}

		// 组合签名
		combinedSig := cc.PK.CombineShares(sigs)
		if combinedSig == nil {
			return
		}

		// 验证组合签名
		if !cc.PK.VerifySignature(combinedSig, messageBytes) {
			return
		}

		// 计算硬币值
		coinValue := cc.computeCoinValue(combinedSig)

		// 发送到输出队列
		if cc.outputQueues[round] != nil {
			select {
			case cc.outputQueues[round] <- coinValue:
			default:
			}
		}
	}
}

// 计算硬币值
func (cc *CommonCoin) computeCoinValue(sig *Point) bool {
	// 使用签名的哈希值的最低有效位（0或1）作为硬币值
	hash := sha256.Sum256(append(sig.X.Bytes(), sig.Y.Bytes()...))
	return (hash[len(hash)-1] & 1) == 1
}

// 获取硬币值
func (cc *CommonCoin) GetCoin(round uint64) bool {
	cc.lock.Lock()

	// 创建输出队列
	if cc.outputQueues[round] == nil {
		cc.outputQueues[round] = make(chan bool, 1)
	}

	// 广播自己的签名份额
	message := fmt.Sprintf("coin_%d_%d", cc.ID, round)
	messageBytes := []byte(message)
	sig := cc.SK.Sign(messageBytes)

	cc.lock.Unlock()

	// 处理自己的签名
	cc.handleMessage(cc.ID, round, sig)

	// 等待结果
	select {
	case result := <-cc.outputQueues[round]:
		return result
	}
}

// 广播签名份额
func (cc *CommonCoin) BroadcastShare(round uint64) *CoinShareMessage {
	message := fmt.Sprintf("coin_%d_%d", cc.ID, round)
	messageBytes := []byte(message)
	sig := cc.SK.Sign(messageBytes)

	return &CoinShareMessage{
		Round: round,
		Sig:   sig,
	}
}

// 处理接收到的签名份额
func (cc *CommonCoin) HandleShare(senderID uint64, msg *CoinShareMessage) {
	cc.messageCh <- coinMessage{
		senderID: senderID,
		round:    msg.Round,
		sig:      msg.Sig,
	}
}

// 硬币份额消息
type CoinShareMessage struct {
	Round uint64
	Sig   *Point
}

// 关闭公共硬币
func (cc *CommonCoin) Close() {
	close(cc.closeCh)
}
