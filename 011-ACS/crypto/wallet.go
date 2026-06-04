package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"log"
	"math/big"

	"golang.org/x/crypto/ripemd160"
)

const version byte = 0x00
const addressChecksumLen = 4

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

// 创建新钱包
func NewWallet() *Wallet {
	privateKey, publicKey := newKeyPair()
	return &Wallet{privateKey, publicKey}
}

// 通过私钥产生公钥
func newKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}

	pubKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
	return *private, pubKey
}

// GobEncode 自定义序列化：只保存私钥 D 字节和公钥字节，避免序列化无导出字段的椭圆曲线类型
func (w Wallet) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(w.PrivateKey.D.Bytes()); err != nil {
		return nil, err
	}
	if err := enc.Encode(w.PublicKey); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode 自定义反序列化：从字节重建私钥和曲线
func (w *Wallet) GobDecode(data []byte) error {
	dec := gob.NewDecoder(bytes.NewReader(data))
	var dBytes []byte
	if err := dec.Decode(&dBytes); err != nil {
		return err
	}
	var pubKey []byte
	if err := dec.Decode(&pubKey); err != nil {
		return err
	}
	curve := elliptic.P256()
	w.PrivateKey.D = new(big.Int).SetBytes(dBytes)
	w.PrivateKey.PublicKey.Curve = curve
	w.PrivateKey.PublicKey.X, w.PrivateKey.PublicKey.Y = curve.ScalarBaseMult(dBytes)
	w.PublicKey = pubKey
	return nil
}

// 验证地址是否有效
func IsValidForAddress(address []byte) bool {
	version_public_checksumBytes := Base58Decode(address)

	checksumBytes := version_public_checksumBytes[len(version_public_checksumBytes)-addressChecksumLen:]
	version_ripemd160Hash := version_public_checksumBytes[:len(version_public_checksumBytes)-addressChecksumLen]
	checkBytes := CheckSum(version_ripemd160Hash)
	return bytes.Equal(checksumBytes, checkBytes)
}

// 返回钱包地址
func (w *Wallet) GetAddress() []byte {
	ripemd160Hash := Ripemd160Hash(w.PublicKey)
	version_ripemd160Hash := append([]byte{version}, ripemd160Hash...)
	checksumBytes := CheckSum(version_ripemd160Hash)
	bytes := append(version_ripemd160Hash, checksumBytes...)
	address := Base58Encode(bytes)
	return address
}

// CheckSum 计算校验和
func CheckSum(payload []byte) []byte {
	hash1 := sha256.Sum256(payload)
	hash2 := sha256.Sum256(hash1[:])
	return hash2[:addressChecksumLen]
}

// Ripemd160Hash 计算公钥的 RIPEMD160 哈希
func Ripemd160Hash(publicKey []byte) []byte {
	// 256 哈希
	hash256 := sha256.New()
	hash256.Write(publicKey)
	hash := hash256.Sum(nil)

	// RIPEMD160 哈希
	ripemd160 := ripemd160.New()
	ripemd160.Write(hash)

	return ripemd160.Sum(nil)
}
