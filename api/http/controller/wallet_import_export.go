package controller

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/gin-gonic/gin"
	"github.com/mr-tron/base58"
	"github.com/okx/go-wallet-sdk/coins/ethereum"
	"github.com/okx/go-wallet-sdk/coins/solana/base"

	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/codes"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/store"
	"github.com/hellodex/HelloSecurity/system"
	"github.com/hellodex/HelloSecurity/wallet"
	"github.com/hellodex/HelloSecurity/wallet/enc"
)

// 导入钱包请求
// 调用链路: TGBot导入 -> API透传 -> Security
type importWalletPKReq struct {
	UUID        string `json:"uuid"`
	WalletId    string `json:"walletId"`
	ChainCode   string `json:"chainCode"`
	EncryptedPK string `json:"encryptedPK"`
}

// 导出钱包请求
// 调用链路: TGBot导出 -> API透传 -> Security
type exportWalletPKReq struct {
	UUID      string `json:"uuid"`
	WalletId  string `json:"walletId"`
	WalletKey string `json:"walletKey"`
}

// ImportWalletPK 导入钱包私钥
// 调用链路: TGBot导入 -> API透传 -> Security -> 验证/存储 -> 返回钱包信息
func ImportWalletPK(c *gin.Context) {
	var req importWalletPKReq
	res := common.Response{}
	res.Timestamp = time.Now().Unix()

	if err := c.ShouldBindJSON(&req); err != nil {
		mylog.Infof("ImportWalletPK 请求解析失败: %v", err)
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "Invalid request"
		c.JSON(http.StatusOK, res)
		return
	}

	// 参数校验
	if req.UUID == "" || req.WalletId == "" || req.ChainCode == "" || req.EncryptedPK == "" {
		res.Code = codes.CODE_ERR_BAT_PARAMS
		res.Msg = "参数不完整"
		c.JSON(http.StatusOK, res)
		return
	}

	// 校验链是否支持
	supp, evm := wallet.IsSupp(wallet.ChainCode(req.ChainCode))
	if !supp {
		res.Code = codes.CODE_ERR_BAT_PARAMS
		res.Msg = "不支持的链"
		c.JSON(http.StatusOK, res)
		return
	}

	db := system.GetDb()

	// 查 wallet_keys 获取 walletKey（用默认钱包的 walletId + uuid 查询）
	var wk model.WalletKeys
	err := db.Table("wallet_keys").
		Where("wallet_id = ? AND user_id = ?", req.WalletId, req.UUID).
		First(&wk).Error
	if err != nil {
		mylog.Infof("ImportWalletPK wallet_key查询失败, walletId=%s, uuid=%s, err=%v", req.WalletId, req.UUID, err)
		res.Code = codes.CODE_ERR_AUTH_FAIL
		res.Msg = "walletKey无效"
		c.JSON(http.StatusOK, res)
		return
	}

	// 派生传输解密密钥
	aesKey := enc.DeriveTransportKey(req.UUID, wk.WalletKey)

	// 解密得到明文私钥
	plainPK, err := enc.DecryptTransport(aesKey, req.EncryptedPK)
	if err != nil {
		mylog.Infof("ImportWalletPK 传输解密失败, uuid=%s, err=%v", req.UUID, err)
		res.Code = codes.CODE_ERR
		res.Msg = "私钥解密失败"
		c.JSON(http.StatusOK, res)
		return
	}

	plainPKStr := string(plainPK)
	var address string

	// 私钥 -> 地址验证
	if evm {
		// EVM: 明文是 hex 字符串
		address, err = deriveEVMAddress(plainPKStr)
		if err != nil {
			mylog.Infof("ImportWalletPK EVM私钥验证失败, uuid=%s, err=%v", req.UUID, err)
			res.Code = codes.CODE_ERR_BAT_PARAMS
			res.Msg = "无效的EVM私钥"
			c.JSON(http.StatusOK, res)
			return
		}
	} else {
		// Solana: 明文是 base58 字符串
		address, err = deriveSolanaAddress(plainPKStr)
		if err != nil {
			mylog.Infof("ImportWalletPK Solana私钥验证失败, uuid=%s, err=%v", req.UUID, err)
			res.Code = codes.CODE_ERR_BAT_PARAMS
			res.Msg = "无效的Solana私钥"
			c.JSON(http.StatusOK, res)
			return
		}
	}

	// 防重复导入
	var existCount int64
	if evm {
		// EVM 地址大小写混合，必须 LOWER 比较
		db.Model(&model.WalletGenerated{}).
			Where("LOWER(wallet) = LOWER(?) AND chain_code = ? AND status = ? AND user_id = ?", address, req.ChainCode, "00", req.UUID).
			Count(&existCount)
	} else {
		// Solana base58 区分大小写，原样比较
		db.Model(&model.WalletGenerated{}).
			Where("wallet = ? AND chain_code = ? AND status = ? AND user_id = ?", address, req.ChainCode, "00", req.UUID).
			Count(&existCount)
	}
	if existCount > 0 {
		res.Code = codes.CODE_ERR_EXIST_OBJ
		res.Msg = "钱包已导入"
		c.JSON(http.StatusOK, res)
		return
	}

	// 创建 WalletGroup（无助记词组, source=10）
	importGroup := &model.WalletGroup{
		UserID:         req.UUID,
		CreateTime:     time.Now(),
		EncryptMem:     "",
		EncryptVersion: "AES:1",
		Nonce:          0,
		VaultType:      0,
		Source:         10,
	}
	if err := db.Save(importGroup).Error; err != nil {
		mylog.Errorf("ImportWalletPK 创建WalletGroup失败, uuid=%s, err=%v", req.UUID, err)
		res.Code = codes.CODE_ERR
		res.Msg = "创建钱包组失败"
		c.JSON(http.StatusOK, res)
		return
	}

	// 存储加密（用 Shamir 主密钥 AES-GCM 加密，与注册钱包一致）
	var storagePlain string
	if evm {
		// EVM: hex 字符串直接存储
		storagePlain = plainPKStr
	} else {
		// Solana: base58 转 base64 再存储（与 GenerateSolana sec.go 一致）
		raw64, _ := base58.Decode(plainPKStr)
		storagePlain = base64.StdEncoding.EncodeToString(raw64)
	}

	encPKBytes, err := enc.Porter().Encrypt([]byte(storagePlain))
	if err != nil {
		mylog.Errorf("ImportWalletPK 存储加密失败, uuid=%s, err=%v", req.UUID, err)
		res.Code = codes.CODE_ERR
		res.Msg = "私钥加密失败"
		c.JSON(http.StatusOK, res)
		return
	}
	encPKBase64 := base64.StdEncoding.EncodeToString(encPKBytes)

	// 创建 WalletGenerated（source=10, nonce=12）
	wg := model.WalletGenerated{
		UserID:         req.UUID,
		ChainCode:      req.ChainCode,
		Wallet:         address,
		EncryptPK:      encPKBase64,
		EncryptVersion: "AES",
		CreateTime:     time.Now(),
		Channel:        "primary",
		CanPort:        false,
		Status:         "00",
		GroupID:        importGroup.ID,
		Nonce:          12,
		Source:         10,
	}
	if err := db.Save(&wg).Error; err != nil {
		mylog.Errorf("ImportWalletPK 创建WalletGenerated失败, uuid=%s, err=%v", req.UUID, err)
		res.Code = codes.CODE_ERR
		res.Msg = "创建钱包记录失败"
		c.JSON(http.StatusOK, res)
		return
	}

	// 生成 WalletKey（临时交易凭证）
	walletKey := common.MyIDStr()
	wkNew := model.WalletKeys{
		WalletKey:  walletKey,
		WalletId:   wg.ID,
		Channel:    "primary",
		ExpireTime: time.Now().AddDate(10, 0, 0).Unix(),
		UserId:     req.UUID,
	}
	if err := db.Create(&wkNew).Error; err != nil {
		mylog.Errorf("ImportWalletPK 创建WalletKey失败, uuid=%s, err=%v", req.UUID, err)
	}

	// 生成 TaskWalletKey（跟单密钥）
	uuidInt, parseErr := strconv.ParseInt(req.UUID, 10, 64)
	if parseErr == nil && uuidInt > 0 {
		existingKey, _ := store.TaskWalletKeyGetByUuidAndWallet(uuidInt, wg.ID)
		if existingKey == nil {
			newKey := common.MyIDStr()
			tk := model.TaskWalletKeys{
				UUID:          uuidInt,
				WalletID:      wg.ID,
				TaskWalletKey: newKey,
			}
			if saveErr := store.TaskWalletKeySave(tk); saveErr != nil {
				mylog.Infof("ImportWalletPK 创建taskWalletKey失败, uuid=%d, walletId=%d, err=%v", uuidInt, wg.ID, saveErr)
			}
		}
	}

	// 返回结果
	res.Code = codes.CODE_SUCCESS
	res.Msg = "success"
	res.Data = gin.H{
		"groupId":   importGroup.ID,
		"walletId":  wg.ID,
		"walletKey": walletKey,
		"address":   address,
		"chainCode": req.ChainCode,
	}
	c.JSON(http.StatusOK, res)
}

// ExportWalletPK 导出钱包私钥
// 调用链路: TGBot导出 -> API透传 -> Security -> 解密/加密 -> 返回密文
func ExportWalletPK(c *gin.Context) {
	var req exportWalletPKReq
	res := common.Response{}
	res.Timestamp = time.Now().Unix()

	if err := c.ShouldBindJSON(&req); err != nil {
		mylog.Infof("ExportWalletPK 请求解析失败: %v", err)
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "Invalid request"
		c.JSON(http.StatusOK, res)
		return
	}

	// 参数校验
	if req.UUID == "" || req.WalletId == "" || req.WalletKey == "" {
		res.Code = codes.CODE_ERR_BAT_PARAMS
		res.Msg = "ExportWalletPK-参数不完整"
		c.JSON(http.StatusOK, res)
		return
	}

	db := system.GetDb()

	// 验证 wallet_keys: wallet_id + user_id + wallet_key 三重校验
	var wk model.WalletKeys
	err := db.Table("wallet_keys").
		Where("wallet_id = ? AND user_id = ? AND wallet_key = ?", req.WalletId, req.UUID, req.WalletKey).
		First(&wk).Error
	if err != nil {
		mylog.Infof("ExportWalletPK wallet_key校验失败, walletId=%s, uuid=%s, err=%v", req.WalletId, req.UUID, err)
		res.Code = codes.CODE_ERR_AUTH_FAIL
		res.Msg = "ExportWalletPK-钱包凭证无效"
		c.JSON(http.StatusOK, res)
		return
	}

	// 验证 walletId 归属 uuid（防越权）
	var wg model.WalletGenerated
	err = db.Model(&model.WalletGenerated{}).Where("id = ? AND status = ?", req.WalletId, "00").First(&wg).Error
	if err != nil {
		res.Code = codes.CODE_ERR_AUTH_FAIL
		res.Msg = "钱包不存在"
		c.JSON(http.StatusOK, res)
		return
	}
	if wg.UserID != req.UUID {
		res.Code = codes.CODE_ERR_AUTH_FAIL
		res.Msg = "用户无权操作此钱包"
		c.JSON(http.StatusOK, res)
		return
	}

	// 解密存储的私钥（Shamir 主密钥解密）
	encryptedPKBytes, err := base64.StdEncoding.DecodeString(wg.EncryptPK)
	if err != nil {
		mylog.Errorf("ExportWalletPK base64解码失败, walletId=%s, err=%v", req.WalletId, err)
		res.Code = codes.CODE_ERR
		res.Msg = "私钥数据异常"
		c.JSON(http.StatusOK, res)
		return
	}
	nonceSize := wg.Nonce
	if len(encryptedPKBytes) < nonceSize {
		res.Code = codes.CODE_ERR
		res.Msg = "私钥数据异常"
		c.JSON(http.StatusOK, res)
		return
	}
	nonce := encryptedPKBytes[:nonceSize]
	cipherData := encryptedPKBytes[nonceSize:]

	storagePlain, err := enc.Porter().Decrypt(cipherData, nonce)
	if err != nil {
		mylog.Errorf("ExportWalletPK 存储解密失败, walletId=%s, err=%v", req.WalletId, err)
		res.Code = codes.CODE_ERR
		res.Msg = "私钥解密失败"
		c.JSON(http.StatusOK, res)
		return
	}

	// 转回用户输入格式（传输明文）
	var transportPlain string
	_, evm := wallet.IsSupp(wallet.ChainCode(wg.ChainCode))
	if evm {
		// EVM: 存储就是 hex 字符串，直接使用
		transportPlain = string(storagePlain)
	} else {
		// Solana: 存储是 base64 字符串，需转回 base58
		raw64, err := base64.StdEncoding.DecodeString(string(storagePlain))
		if err != nil {
			mylog.Errorf("ExportWalletPK Solana base64解码失败, walletId=%s, err=%v", req.WalletId, err)
			res.Code = codes.CODE_ERR
			res.Msg = "私钥格式异常"
			c.JSON(http.StatusOK, res)
			return
		}
		transportPlain = base58.Encode(raw64)
	}

	// 传输加密: uuid + walletKey 派生密钥
	aesKey := enc.DeriveTransportKey(req.UUID, req.WalletKey)
	encryptedPK, err := enc.EncryptTransport(aesKey, []byte(transportPlain))
	if err != nil {
		mylog.Errorf("ExportWalletPK 传输加密失败, walletId=%s, err=%v", req.WalletId, err)
		res.Code = codes.CODE_ERR
		res.Msg = "私钥加密失败"
		c.JSON(http.StatusOK, res)
		return
	}

	// 返回结果（不落库，纯内存操作）
	res.Code = codes.CODE_SUCCESS
	res.Msg = "success"
	res.Data = gin.H{
		"walletId":    req.WalletId,
		"encryptedPK": encryptedPK,
		"chainCode":   wg.ChainCode,
	}
	c.JSON(http.StatusOK, res)
}

// deriveEVMAddress 从 EVM 私钥 hex 字符串派生地址
// 调用链路: ImportWalletPK -> 本方法
func deriveEVMAddress(privateKeyHex string) (string, error) {
	// 去掉 0x 前缀
	hexStr := strings.TrimPrefix(privateKeyHex, "0x")
	pkBytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", fmt.Errorf("hex解码失败: %w", err)
	}
	if len(pkBytes) != 32 {
		return "", fmt.Errorf("私钥长度不正确: %d", len(pkBytes))
	}
	prvKey, _ := btcec.PrivKeyFromBytes(pkBytes)
	address := ethereum.GetNewAddress(prvKey.PubKey())
	if address == "" || address == "0x0000000000000000000000000000000000000000" {
		return "", fmt.Errorf("无效的EVM地址")
	}
	return address, nil
}

// deriveSolanaAddress 从 Solana 私钥 base58 字符串派生地址
// 调用链路: ImportWalletPK -> 本方法
func deriveSolanaAddress(privateKeyBase58 string) (string, error) {
	privKey, err := base.PrivateKeyFromBase58(privateKeyBase58)
	if err != nil {
		return "", fmt.Errorf("不是有效的Base58私钥: %w", err)
	}
	if len(privKey) != 64 {
		return "", fmt.Errorf("私钥长度不正确: %d (需要64字节)", len(privKey))
	}
	address := privKey.PublicKey().String()
	if address == "" || address == "11111111111111111111111111111111" {
		return "", fmt.Errorf("无效的Solana地址")
	}
	return address, nil
}
