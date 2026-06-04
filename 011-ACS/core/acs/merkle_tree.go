package ACS

import (
	"bytes"
	"crypto/sha256"
	"fmt"
)

// 简化的默克尔树实现
type MerkleTree struct {
	leaves [][]byte
	tree   [][]byte
}

// 创建默克尔树
func NewMerkleTree(leaves [][]byte) *MerkleTree {
	mt := &MerkleTree{
		leaves: leaves,
	}
	mt.buildTree()
	return mt
}

// 构建默克尔树
func (mt *MerkleTree) buildTree() {
	n := len(mt.leaves)
	if n == 0 {
		return
	}

	// 计算树的高度
	height := 0
	for (1 << height) < n {
		height++
	}
	treeSize := (1 << (height + 1)) - 1
	mt.tree = make([][]byte, treeSize)

	// 填充叶子节点
	leafStart := (1 << height) - 1
	for i, leaf := range mt.leaves {
		mt.tree[leafStart+i] = leaf
	}

	// 构建内部节点
	for level := height - 1; level >= 0; level-- {
		levelStart := (1 << level) - 1
		levelEnd := (1 << (level + 1)) - 1
		for i := levelStart; i < levelEnd; i++ {
			left := 2*i + 1
			right := 2*i + 2
			if left < len(mt.tree) && right < len(mt.tree) {
				mt.tree[i] = mt.hashChildren(mt.tree[left], mt.tree[right])
			}
		}
	}
}

// 哈希子节点
func (mt *MerkleTree) hashChildren(left, right []byte) []byte {
	hasher := sha256.New()
	hasher.Write(left)
	hasher.Write(right)
	return hasher.Sum(nil)
}

// 获取根哈希
func (mt *MerkleTree) Root() []byte {
	if len(mt.tree) == 0 {
		return nil
	}
	return mt.tree[0]
}

// 获取指定索引的证明路径
func (mt *MerkleTree) GetProof(index int) ([][]byte, error) {
	if index < 0 || index >= len(mt.leaves) {
		return nil, fmt.Errorf("invalid index")
	}

	var proof [][]byte
	leafStart := (1 << (mt.getHeight())) - 1
	current := leafStart + index

	for current > 0 {
		sibling := current ^ 1 // 异或操作获取兄弟节点
		if sibling < len(mt.tree) {
			proof = append(proof, mt.tree[sibling])
		}
		current = (current - 1) / 2
	}

	return proof, nil
}

// 获取树的高度
func (mt *MerkleTree) getHeight() int {
	if len(mt.leaves) == 0 {
		return 0
	}
	height := 0
	for (1 << height) < len(mt.leaves) {
		height++
	}
	return height
}

// 验证默克尔证明
func VerifyMerkleProof(leaf []byte, proof [][]byte, root []byte, index int) bool {
	current := sha256.Sum256(leaf)

	for i, sibling := range proof {
		hasher := sha256.New()
		if (index>>i)&1 == 0 {
			// 当前节点是左子节点
			hasher.Write(current[:])
			hasher.Write(sibling)
		} else {
			// 当前节点是右子节点
			hasher.Write(sibling)
			hasher.Write(current[:])
		}
		hash := hasher.Sum(nil)
		copy(current[:], hash)
	}

	return bytes.Equal(current[:], root)
}

// 简化的默克尔树验证函数，用于RBC
func VerifyMerkleProofSimple(leaf []byte, proof [][]byte, root []byte, index int) bool {
	if len(proof) == 0 {
		return false
	}
	return VerifyMerkleProof(leaf, proof, root, index)
}
