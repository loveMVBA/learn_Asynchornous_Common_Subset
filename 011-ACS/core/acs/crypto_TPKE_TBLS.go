package ACS

/*
总的来说，这个go文件就是实现TPKE和TBLS现这两类密码学组件
*/

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"math/big"
	"sort"
)

// 简化的椭圆曲线点结构。
// 本文件统一使用 Go 标准库的 P-256 曲线。
type Point struct {
	X *big.Int
	Y *big.Int
}

// 阈值加密公钥
type TPKEPublicKey struct {
	N   int      // 总节点数
	K   int      // 阈值
	VK  *Point   // 全局验证密钥 / 公钥 g^secret
	VKs []*Point // 各节点的验证密钥 g^sk_i
}

// 阈值加密私钥
type TPKEPrivateKey struct {
	*TPKEPublicKey
	I  int      // 节点索引，0-based
	SK *big.Int // 私钥份额
}

// 加密密文
type Ciphertext struct {
	U *Point // g^r
	V []byte // message XOR H(VK^r)，当前实现要求 message 为 32 字节
	W *Point // CCA 证明占位点：H(U,V)^r
}

// 解密份额
type DecryptionShare struct {
	U_i *Point // U^sk_i
}

func ecCurve() elliptic.Curve {
	return elliptic.P256()
}

func ecOrder() *big.Int {
	return new(big.Int).Set(ecCurve().Params().N)
}

func normalizeScalar(k *big.Int) *big.Int {
	if k == nil {
		return big.NewInt(0)
	}
	order := ecOrder()
	out := new(big.Int).Mod(new(big.Int).Set(k), order)
	if out.Sign() < 0 {
		out.Add(out, order)
	}
	return out
}

func randScalar() (*big.Int, error) {
	order := ecOrder()
	for {
		k, err := rand.Int(rand.Reader, order)
		if err != nil {
			return nil, err
		}
		if k.Sign() != 0 {
			return k, nil
		}
	}
}

func NewPoint(x, y *big.Int) *Point {
	if x == nil || y == nil {
		return &Point{}
	}
	return &Point{
		X: new(big.Int).Set(x),
		Y: new(big.Int).Set(y),
	}
}

func InfinityPoint() *Point {
	return &Point{}
}

func BasePoint() *Point {
	curve := ecCurve()
	params := curve.Params()
	return NewPoint(params.Gx, params.Gy)
}

func BaseScalarMult(k *big.Int) *Point {
	k = normalizeScalar(k)
	if k.Sign() == 0 {
		return InfinityPoint()
	}

	x, y := ecCurve().ScalarBaseMult(k.Bytes())
	return NewPoint(x, y)
}

func (p *Point) IsInfinity() bool {
	return p == nil || p.X == nil || p.Y == nil
}

func (p *Point) IsOnCurve() bool {
	if p.IsInfinity() {
		return true
	}
	return ecCurve().IsOnCurve(p.X, p.Y)
}

func (p *Point) Clone() *Point {
	if p.IsInfinity() {
		return InfinityPoint()
	}
	return NewPoint(p.X, p.Y)
}

func (p *Point) Equal(q *Point) bool {
	if p.IsInfinity() || q.IsInfinity() {
		return p.IsInfinity() && q.IsInfinity()
	}
	return p.X.Cmp(q.X) == 0 && p.Y.Cmp(q.Y) == 0
}

func (p *Point) Add(q *Point) *Point {
	if p.IsInfinity() {
		return q.Clone()
	}
	if q.IsInfinity() {
		return p.Clone()
	}

	curve := ecCurve()
	if !curve.IsOnCurve(p.X, p.Y) || !curve.IsOnCurve(q.X, q.Y) {
		return nil
	}

	x, y := curve.Add(p.X, p.Y, q.X, q.Y)
	return NewPoint(x, y)
}

func (p *Point) ScalarMult(k *big.Int) *Point {
	if p.IsInfinity() {
		return InfinityPoint()
	}
	if !p.IsOnCurve() {
		return nil
	}

	k = normalizeScalar(k)
	if k.Sign() == 0 {
		return InfinityPoint()
	}

	x, y := ecCurve().ScalarMult(p.X, p.Y, k.Bytes())
	return NewPoint(x, y)
}

func serializePoint(p *Point) []byte {
	if p.IsInfinity() {
		return []byte{0}
	}
	return elliptic.Marshal(ecCurve(), p.X, p.Y)
}

func hashPoint(p *Point) [32]byte {
	return sha256.Sum256(serializePoint(p))
}

// HashToPoint 将任意消息确定性映射到 P-256 曲线上的一个点。
func HashToPoint(message []byte) *Point {
	curve := ecCurve()
	params := curve.Params()
	p := params.P
	b := params.B
	three := big.NewInt(3)

	for counter := uint64(0); ; counter++ {
		input := make([]byte, 0, len(message)+8)
		input = append(input, message...)
		for shift := 56; shift >= 0; shift -= 8 {
			input = append(input, byte(counter>>uint(shift)))
		}

		digest := sha256.Sum256(input)
		x := new(big.Int).SetBytes(digest[:])
		x.Mod(x, p)

		// P-256: y^2 = x^3 - 3x + B mod P
		rhs := new(big.Int).Exp(x, big.NewInt(3), p)
		minus3x := new(big.Int).Mul(three, x)
		rhs.Sub(rhs, minus3x)
		rhs.Add(rhs, b)
		rhs.Mod(rhs, p)

		y := new(big.Int).ModSqrt(rhs, p)
		if y != nil && curve.IsOnCurve(x, y) {
			return NewPoint(x, y)
		}
	}
}

/* 生成TPKE阈值加密密钥对
1. 随机生成 k 阶 Shamir 多项式
2. 多项式常数项 secret 作为主秘密
3. 对每个节点 i 计算 SK_i = f(i+1)
4. 对每个 SK_i 计算验证公钥 VK_i = g^SK_i
5. 全局公钥 VK = g^secret
6. 返回：一个公共 TPKEPublicKey 和 n 个 TPKEPrivateKey 私钥份额
*/

func GenerateTPKEKeys(n, k int) (*TPKEPublicKey, []*TPKEPrivateKey, error) {
	if n <= 0 {
		return nil, nil, errors.New("n must be positive")
	}
	if k <= 0 {
		return nil, nil, errors.New("threshold k must be positive")
	}
	if k > n {
		return nil, nil, errors.New("threshold k cannot be greater than n")
	}

	coefficients, err := randomPolynomial(k)
	if err != nil {
		return nil, nil, err
	}
	secret := coefficients[0]

	SKs := make([]*big.Int, n)
	VKs := make([]*Point, n)
	for i := 0; i < n; i++ {
		x := big.NewInt(int64(i + 1))
		SKs[i] = evaluatePolynomial(coefficients, x)
		VKs[i] = BaseScalarMult(SKs[i])
	}

	pk := &TPKEPublicKey{
		N:   n,
		K:   k,
		VK:  BaseScalarMult(secret),
		VKs: VKs,
	}

	privKeys := make([]*TPKEPrivateKey, n)
	for i := 0; i < n; i++ {
		privKeys[i] = &TPKEPrivateKey{
			TPKEPublicKey: pk,
			I:             i,
			SK:            SKs[i],
		}
	}

	return pk, privKeys, nil
}

func randomPolynomial(degree int) ([]*big.Int, error) {
	coefficients := make([]*big.Int, degree)
	for i := range coefficients {
		coeff, err := randScalar()
		if err != nil {
			return nil, err
		}
		coefficients[i] = coeff
	}
	return coefficients, nil
}

// 多项式求值，所有运算都在曲线阶 N 上进行。
func evaluatePolynomial(coefficients []*big.Int, x *big.Int) *big.Int {
	order := ecOrder()
	result := big.NewInt(0)
	power := big.NewInt(1)
	x = normalizeScalar(x)

	for _, coeff := range coefficients {
		term := new(big.Int).Mul(normalizeScalar(coeff), power)
		term.Mod(term, order)
		result.Add(result, term)
		result.Mod(result, order)

		power.Mul(power, x)
		power.Mod(power, order)
	}

	return result
}

func lagrangeCoefficient(S []int, j int) *big.Int {
	order := ecOrder()
	xj := big.NewInt(int64(j + 1))
	num := big.NewInt(1)
	den := big.NewInt(1)

	for _, jj := range S {
		if jj == j {
			continue
		}

		xm := big.NewInt(int64(jj + 1))

		// num *= -xm
		num.Mul(num, new(big.Int).Neg(xm))
		num.Mod(num, order)

		// den *= xj - xm
		diff := new(big.Int).Sub(xj, xm)
		den.Mul(den, diff)
		den.Mod(den, order)
	}

	denInv := new(big.Int).ModInverse(den, order)
	if denInv == nil {
		return big.NewInt(0)
	}

	lambda := new(big.Int).Mul(num, denInv)
	lambda.Mod(lambda, order)
	return lambda
}

func selectShareIndexes[T any](shares map[int]T, k int) ([]int, error) {
	if len(shares) < k {
		return nil, errors.New("insufficient shares")
	}

	indexes := make([]int, 0, len(shares))
	for i := range shares {
		indexes = append(indexes, i)
	}
	sort.Ints(indexes)
	return indexes[:k], nil
}

// 拉格朗日插值系数
func (pk *TPKEPublicKey) lagrange(S []int, j int) *big.Int {
	return lagrangeCoefficient(S, j)
}

// 加密。当前实现加密 32 字节对称密钥，通常再用该密钥加密真实交易数据。
func (pk *TPKEPublicKey) Encrypt(m []byte) (*Ciphertext, error) {
	if pk == nil || pk.VK == nil || !pk.VK.IsOnCurve() || pk.VK.IsInfinity() {
		return nil, errors.New("invalid public key")
	}
	if len(m) != 32 {
		return nil, errors.New("message must be 32 bytes")
	}

	r, err := randScalar()
	if err != nil {
		return nil, err
	}

	U := BaseScalarMult(r)
	shared := pk.VK.ScalarMult(r)
	mask := hashPoint(shared)

	V := make([]byte, 32)
	for i := range V {
		V[i] = m[i] ^ mask[i]
	}

	hInput := append(serializePoint(U), V...)
	W := HashToPoint(hInput).ScalarMult(r)

	return &Ciphertext{U: U, V: V, W: W}, nil
}

// 生成解密份额
func (sk *TPKEPrivateKey) DecryptShare(ct *Ciphertext) *DecryptionShare {
	if sk == nil || sk.SK == nil || ct == nil || ct.U == nil || !ct.U.IsOnCurve() {
		return nil
	}
	return &DecryptionShare{U_i: ct.U.ScalarMult(sk.SK)}
}

// 验证解密份额。
// 注意：真正的 TPKE 份额验证通常需要双线性配对或额外的 NIZK 证明。
// 标准库 P-256 没有配对能力，因此这里做基础曲线合法性检查。
func (pk *TPKEPublicKey) VerifyShare(i int, share *DecryptionShare, ct *Ciphertext) bool {
	if pk == nil || i < 0 || i >= pk.N || ct == nil || ct.U == nil || share == nil || share.U_i == nil {
		return false
	}
	return ct.U.IsOnCurve() && share.U_i.IsOnCurve() && !share.U_i.IsInfinity()
}

// 组合解密份额
func (pk *TPKEPublicKey) CombineShares(ct *Ciphertext, shares map[int]*DecryptionShare) ([]byte, error) {
	if pk == nil || ct == nil || len(ct.V) != 32 {
		return nil, errors.New("invalid ciphertext")
	}

	S, err := selectShareIndexes(shares, pk.K)
	if err != nil {
		return nil, err
	}

	result := InfinityPoint()
	for _, j := range S {
		share := shares[j]
		if !pk.VerifyShare(j, share, ct) {
			return nil, errors.New("invalid decryption share")
		}

		lambda := pk.lagrange(S, j)
		term := share.U_i.ScalarMult(lambda)
		result = result.Add(term)
		if result == nil {
			return nil, errors.New("failed to combine shares")
		}
	}

	mask := hashPoint(result)
	plaintext := make([]byte, 32)
	for i := range plaintext {
		plaintext[i] = ct.V[i] ^ mask[i]
	}

	return plaintext, nil
}

// AES加密/解密
func AESEncrypt(key, plaintext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	padded := pkcs7Padding(plaintext, aes.BlockSize)

	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	ciphertext := make([]byte, len(padded))
	mode.CryptBlocks(ciphertext, padded)

	return append(iv, ciphertext...), nil
}

func AESDecrypt(key, ciphertext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes")
	}
	if len(ciphertext) < aes.BlockSize || len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("invalid ciphertext length")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	return pkcs7Unpadding(plaintext)
}

func pkcs7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := make([]byte, padding)
	for i := range padtext {
		padtext[i] = byte(padding)
	}
	return append(data, padtext...)
}

func pkcs7Unpadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("empty plaintext")
	}

	padding := int(data[length-1])
	if padding == 0 || padding > length || padding > aes.BlockSize {
		return nil, errors.New("invalid padding")
	}

	for _, b := range data[length-padding:] {
		if int(b) != padding {
			return nil, errors.New("invalid padding")
		}
	}

	return data[:length-padding], nil
}

// 生成随机字节
func randBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}

// 阈值签名相关结构
type TBLSPublicKey struct {
	N   int
	K   int
	VK  *Point
	VKs []*Point
}

type TBLSPrivateKey struct {
	*TBLSPublicKey
	I  int
	SK *big.Int
}

/*
	生成TBLS阈值签名密钥对

1. 生成一个随机 Shamir 多项式 f(x)
2. 用 f(0) 作为主秘密
3. 给 n 个节点分别计算私钥份额 f(1), f(2), ..., f(n)
4. 为每个私钥份额生成公开验证点
5. 生成整体公共验证密钥
6. 返回公共密钥和所有节点的私钥份额
*/
func GenerateTBLSKeys(n, k int) (*TBLSPublicKey, []*TBLSPrivateKey, error) {
	if n <= 0 {
		return nil, nil, errors.New("n must be positive")
	}
	if k <= 0 {
		return nil, nil, errors.New("threshold k must be positive")
	}
	if k > n {
		return nil, nil, errors.New("threshold k cannot be greater than n")
	}

	coefficients, err := randomPolynomial(k)
	if err != nil {
		return nil, nil, err
	}
	secret := coefficients[0]

	SKs := make([]*big.Int, n)
	VKs := make([]*Point, n)
	for i := 0; i < n; i++ {
		x := big.NewInt(int64(i + 1))
		SKs[i] = evaluatePolynomial(coefficients, x)
		VKs[i] = BaseScalarMult(SKs[i])
	}

	pk := &TBLSPublicKey{
		N:   n,
		K:   k,
		VK:  BaseScalarMult(secret),
		VKs: VKs,
	}

	privKeys := make([]*TBLSPrivateKey, n)
	for i := 0; i < n; i++ {
		privKeys[i] = &TBLSPrivateKey{
			TBLSPublicKey: pk,
			I:             i,
			SK:            SKs[i],
		}
	}

	return pk, privKeys, nil
}

// 签名份额：sig_i = H(message)^sk_i
func (sk *TBLSPrivateKey) Sign(message []byte) *Point {
	if sk == nil || sk.SK == nil {
		return nil
	}
	return HashToPoint(message).ScalarMult(sk.SK)
}

// 验证签名份额。
// 真实 BLS 需要配对检查 e(sig_i, g) == e(H(m), VK_i)。P-256 不支持配对，
// 因此这里仅检查索引、消息和曲线点合法性。
func (pk *TBLSPublicKey) VerifyShare(sig *Point, i int, message []byte) bool {
	if pk == nil || i < 0 || i >= pk.N || sig == nil || len(message) == 0 {
		return false
	}
	return sig.IsOnCurve() && !sig.IsInfinity()
}

// 组合签名份额
func (pk *TBLSPublicKey) CombineShares(sigs map[int]*Point) *Point {
	if pk == nil {
		return nil
	}

	S, err := selectShareIndexes(sigs, pk.K)
	if err != nil {
		return nil
	}

	result := InfinityPoint()
	for _, j := range S {
		sig := sigs[j]
		if sig == nil || !sig.IsOnCurve() || sig.IsInfinity() {
			return nil
		}

		lambda := pk.lagrange(S, j)
		term := sig.ScalarMult(lambda)
		result = result.Add(term)
		if result == nil {
			return nil
		}
	}

	return result
}

// 验证完整签名。
// 同 VerifyShare，这里只能做 P-256 点合法性检查，不能完成真正 BLS 配对验证。
func (pk *TBLSPublicKey) VerifySignature(sig *Point, message []byte) bool {
	if pk == nil || sig == nil || len(message) == 0 {
		return false
	}
	return sig.IsOnCurve() && !sig.IsInfinity()
}

// 拉格朗日插值系数（用于阈值签名）
func (pk *TBLSPublicKey) lagrange(S []int, j int) *big.Int {
	return lagrangeCoefficient(S, j)
}
