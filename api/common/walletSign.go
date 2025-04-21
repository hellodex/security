package common

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	mylog "github.com/hellodex/HelloSecurity/log"
	"github.com/mr-tron/base58"
	"strings"
)

// EtherGenSign 使用私钥对消息生成签名
// 返回[]byte
func EtherGenSign(privateKeyHex string, message string) ([]byte, error) {
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")
	// 1. 将十六进制的私钥解析为 ECDSA 私钥
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("无效的私钥: %v", err)
	}

	// 2. 计算消息的哈希 (EIP-191标准)
	messageHash := crypto.Keccak256Hash([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)))

	// 3. 使用私钥对消息哈希签名
	signature, err := crypto.Sign(messageHash.Bytes(), privateKey)
	if err != nil {
		return nil, fmt.Errorf("签名失败: %v", err)
	}

	return signature, nil
}

// EtherVerifySign 验证以太坊 EIP-191 签名
// 验证十六进制编码的签名字符串
func EtherVerifySign(signatureStr string, message string, expectedAddress string) (bool, error) {
	// ml.Log.Debug("VerifyEtherSign:", signatureStr, message, expectedAddress)
	// 1. 确保签名是 0x 开头的 16 进制字符串
	if !strings.HasPrefix(signatureStr, "0x") {
		signatureStr = "0x" + signatureStr
	}

	// 2. 解码签名字符串为字节数组
	signature, err := hexutil.Decode(signatureStr)
	if err != nil {
		mylog.Errorf("解码签名失败: %v", err)
		return false, err
	}

	// 3. 确保签名的长度为 65 字节 (r + s + v)
	if len(signature) != 65 {
		mylog.Errorf("无效的签名长度: %d, 期望: 65", len(signature))
		return false, fmt.Errorf("无效的签名长度: %d, 期望: 65", len(signature))
	}

	// 4. 将 v 转换为 0/1，如果 v 是 27/28，则减去 27
	if signature[64] >= 27 {
		signature[64] -= 27
	}

	// 5. 计算消息的 EIP-191 哈希
	messageHash := crypto.Keccak256Hash([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)))

	// 6. 使用签名恢复公钥
	publicKey, err := crypto.SigToPub(messageHash.Bytes(), signature)
	if err != nil {
		mylog.Errorf("恢复公钥失败: %v", err)
		return false, err
	}

	// 7. 从恢复的公钥计算地址
	recoveredAddress := crypto.PubkeyToAddress(*publicKey)

	// 8. 检查恢复的地址是否与期望的地址匹配（不区分大小写）
	isVerified := strings.EqualFold(recoveredAddress.Hex(), expectedAddress)
	if !isVerified {
		mylog.Errorf("地址不匹配, 期望: %s, 恢复: %s", expectedAddress, recoveredAddress.Hex())
		//fmt.Printf("地址不匹配, 期望: %s, 恢复: %s\n", expectedAddress, recoveredAddress.Hex())
		return false, fmt.Errorf("地址不匹配, 期望: %s, 恢复: %s", expectedAddress, recoveredAddress.Hex())
	}

	return true, nil
}

// EtherGenSignHex 使用私钥对消息生成签名
// 返回十六进制编码的签名字符串
func EtherGenSignHex(privateKeyHex string, message string) (string, error) {

	signature, err := EtherGenSign(privateKeyHex, message)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(signature), nil
}

// SolanaGenSign 使用私钥对消息进行签名
// 返回base64编码的签名字符串
func SolanaGenSign(message string, privateKeyBase58 string) (string, error) {
	// 解码base58私钥
	privateKey, err := base58.FastBase58Decoding(privateKeyBase58)
	if err != nil {
		fmt.Printf("私钥解析失败: %v", err)
		return "", fmt.Errorf("failed to decode private key: %v", err)
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("invalid private key length: %d", len(privateKey))
	}
	// 签名消息
	signature := ed25519.Sign(privateKey, []byte(message))
	// 将签名转换为base64编码
	signatureBase64 := base64.StdEncoding.EncodeToString(signature)
	return signatureBase64, nil
}

// SolanaVerifySign 验证消息签名
// 验证base64 signature
func SolanaVerifySign(signatureBase64 string, message string, expectedAddress string) (bool, error) {
	publicKey, err := base58.Decode(expectedAddress)
	if err != nil {
		fmt.Printf("钱包地址解析失败: %v", err)
		return false, err
	}
	// 解码base64签名
	signature, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %v", err)
	}
	// 验证签名
	return ed25519.Verify(publicKey, []byte(message), signature), nil
}
