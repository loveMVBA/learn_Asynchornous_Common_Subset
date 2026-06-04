删掉PoW相关的逻辑，替换上ACS, 调整一下依赖，这样我们就实现了一个异步BFT协议 (aBFT) ！！

协议流程如下：

```mermaid
    A[CLI send transaction] --> B[Create signed transaction]
    B --> C[Broadcast TX]
    C --> D[Validators add TX to mempool]

    D --> E[Start ACS epoch]
    E --> F[Each validator proposes local tx batch]
    F --> G[RBC reliable broadcast]
    G --> H[BBA decides which proposals are included]
    H --> I[ACS outputs proposal subset]

    I --> J[Decode proposals]
    J --> K[Sort by proposer ID]
    K --> L[Deduplicate transactions]
    L --> M[Validate signatures and UTXO]
    M --> N[Build deterministic block]
    N --> O[Commit block locally]
    O --> P[Update UTXO set]
    P --> Q[Next epoch]
```
但是Dumbo 的作者们发现影响基于ACS的aBFT性能的一个瓶颈是 ABA。由于在每轮共识中每个节点都要运行 N 个 ABA 的实例，每个实例都要验证 O(N^2) 个阈值签名，这对 CPU 的消耗很大。如下图所示，RBC 的运行时间相比 ABA 几乎可以忽略不计，而且随着 N 增大，运行 ABA 所需要的时间越来越长。

根据这个问题，他们提出了两种完全不同的优化路线：Dumbo1 用小委员会提出候选集合，从而把 ABA 投票次数从 n 个降到 k 个，而 Dumbo2 废除委员会靠“改变投票机制（MVBA）”来一步到位。

然而Dumbo2 废除委员会后，全网所有 N 个节点在 MVBA 的投票阶段都需要深度参与。当 Leader 抛出提案后，全网 N 个节点之间会产生 O(N^2) 级别的控制消息（主要是签名碎片和投票状态）交互。

而且Dumbo2 的ACS依然是串行交替进行的。一轮共识必须等全网把 RBC 广播完、选好子集、再完成门限解密，才能开启下一轮。当全网在等 Leader 摇硬币和投票时，数据传输通道其实是闲置的。而且节点必须打包超大的 Batch，大包在异步网络中传输极慢，导致系统整体延迟直线上升。

我们会在下一趴解决这些问题 ~~