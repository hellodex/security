package model

import (
	"crypto/sha256"
	"fmt"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/shopspring/decimal"
	"time"

	"github.com/hellodex/HelloSecurity/codes"
)

type WalletGenerated struct {
	ID             uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID         string    `gorm:"column:user_id" json:"user_id"`
	Wallet         string    `gorm:"column:wallet" json:"wallet"`
	ChainCode      string    `gorm:"column:chain_code" json:"chain_code"`
	EncryptPK      string    `gorm:"column:encrypt_pk" json:"encrypt_pk"`
	EncryptVersion string    `gorm:"column:encrypt_version" json:"encrypt_version"`
	CreateTime     time.Time `gorm:"column:create_time" json:"create_time"`
	Channel        string    `gorm:"column:channel" json:"channel"`
	CanPort        bool      `gorm:"column:canport" json:"canport"`
	Status         string    `gorm:"column:status" json:"status"`
	GroupID        uint64    `gorm:"column:group_id" json:"group_id"`
	Nonce          int       `gorm:"column:nonce" json:"nonce"`
}

// TableName sets the insert table name for this struct type
func (WalletGenerated) TableName() string {
	return "wallet_generated"
}

type WalletGroup struct {
	ID             uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID         string    `gorm:"column:user_id" json:"user_id"`
	CreateTime     time.Time `gorm:"column:create_time" json:"create_time"`
	EncryptMem     string    `gorm:"column:encrypt_mem" json:"encrypt_mem"`
	EncryptVersion string    `gorm:"column:encrypt_version" json:"encrypt_version"`
	Nonce          int       `gorm:"column:nonce" json:"nonce"`
	VaultType      int       `gorm:"column:vault_type" json:"vault_type"`
}

func (WalletGroup) TableName() string {
	return "wallet_group"
}

type WalletLog struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	WalletID  int64     `gorm:"column:wallet_id" json:"wallet_id"`
	Wallet    string    `gorm:"column:wallet" json:"wallet"`
	Data      string    `gorm:"column:data" json:"data"`
	Sig       string    `gorm:"column:sig" json:"sig"`
	ChainCode string    `gorm:"column:chain_code" json:"chain_code"`
	TxHash    string    `gorm:"column:tx_hash" json:"tx_hash"`
	OpTime    time.Time `gorm:"column:op_time" json:"op_time"`
	Operation string    `gorm:"column:operation" json:"operation"`
	Err       string    `gorm:"column:error" json:"error"`
}

func (WalletLog) TableName() string {
	return "wallet_log"
}

type SysChannel struct {
	ID         uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	AppID      string    `gorm:"column:app_id" json:"app_id"`
	AppKey     string    `gorm:"column:app_key;size:100" json:"app_key"`
	Status     string    `gorm:"column:status" json:"status"`
	SigMethod  string    `gorm:"column:sig_method;size:255" json:"sig_method"`
	CreateTime time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time" json:"update_time"`
}

func (SysChannel) TableName() string {
	return "sys_channel"
}

func (t *SysChannel) Verify(data, sig string) (bool, int) {
	if t.SigMethod != "SHA256" {
		return false, codes.CODE_ERR_SIGMETHOD_UNSUPP
	}
	if len(data) == 0 || len(sig) == 0 {
		return false, codes.CODE_ERR_AUTHTOKEN_FAIL
	}
	data = fmt.Sprintf("%s%s", data, t.AppKey)

	hashByte := sha256.Sum256([]byte(data))
	hash := fmt.Sprintf("%x", hashByte[:])
	if hash != sig {
		return false, codes.CODE_ERR_AUTHTOKEN_FAIL
	}
	return true, codes.CODE_SUCCESS
}

type SysDes struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Desk       string    `gorm:"column:desk" json:"desk"`
	Desv       string    `gorm:"column:desv; json:"desv"`
	UpdateTime time.Time `gorm:"column:update_time" json:"update_time"`
	Flag       int       `gorm:"column:flag; json:"flag"`
}

func (SysDes) TableName() string {
	return "sys_des"
}

type WalletKeys struct {
	UserId     string `gorm:"column:user_id" json:"userId"`
	WalletId   uint64 `gorm:"column:wallet_id" json:"walletId"`
	WalletKey  string `gorm:"column:wallet_key" json:"walletKey"`
	Channel    string `gorm:"column:channel" json:"channel"`
	UserDevice string `gorm:"column:user_device" json:"userDevice"`
	ExpireTime int64  `gorm:"column:expire_time" json:"expireTime"`
}

// 钱包密钥表
func (WalletKeys) TableName() string {
	return "wallet_keys"
}

type LimitKeys struct {
	ID       int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	WalletID uint64 `gorm:"column:wallet_id" json:"walletId"`
	LimitKey string `gorm:"column:limit_key" json:"limitKey"`
}

// 钱包密钥表
func (LimitKeys) TableName() string {
	return "limit_keys"
}

type TgLogin struct {
	ID           int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Token        string `gorm:"column:token" json:"token"`
	AccountID    string `gorm:"column:account_id" json:"tgUserID"`
	GenerateTime int64  `gorm:"column:generate_time" json:"generateTime"`
	ExpireTime   int64  `gorm:"column:expire_time" json:"expireTime"`
	IsUsed       int8   `gorm:"column:is_used" json:"isUsed"`
}

// tg登录信息表
func (TgLogin) TableName() string {
	return "tg_login"
}

/*
CREATE TABLE `auth_account` (
`id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键',
`user_uuid` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0' COMMENT '关联用户uuid',
`account_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '三方账户唯一标识',
`account_type` int DEFAULT NULL COMMENT '用户类型',
`token` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '授权token',
`detail` json DEFAULT NULL COMMENT '详情信息',
`create_time` datetime DEFAULT CURRENT_TIMESTAMP,
`update_time` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
`status` int DEFAULT '0' COMMENT '0:未失效 1: 失效',
`secret_key` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '2fa',
PRIMARY KEY (`id`),
KEY `auth_account_account_id_index` (`account_id`)
) ENGINE=InnoDB AUTO_INCREMENT=754 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='授权表';
*/

type AuthAccount struct {
	ID          int64                      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserUUID    string                     `gorm:"column:user_uuid" json:"userUuid"`
	AccountID   string                     `gorm:"column:account_id" json:"accountId"`
	AccountType int                        `gorm:"column:account_type" json:"accountType"`
	Token       string                     `gorm:"column:token" json:"token"`
	Detail      *string                    `gorm:"column:detail" json:"detail"`
	CreateTime  time.Time                  `gorm:"column:create_time" json:"createTime"`
	UpdateTime  time.Time                  `gorm:"column:update_time" json:"updateTime"`
	Status      int                        `gorm:"column:status" json:"status"` // 0:正常/未失效  1 注销 2 冻结
	SecretKey   string                     `gorm:"column:secret_key" json:"secretKey"`
	Wallets     []common.AuthGetBackWallet `gorm:"-" json:"wallets"`
}

// 授权表
func (AuthAccount) TableName() string {
	return "auth_account"
}

/*CREATE TABLE `user_info` (
  `id` int NOT NULL AUTO_INCREMENT,
  `uuid` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '用户id,由系统生成',
  `user_address` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '用户钱包地址',
  `ip` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'v4/v6 暂不保存',
  `create_time` timestamp NULL DEFAULT (now()),
  `invitation_code` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '邀请码',
  `update_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `user_info_user_address_uindex` (`user_address`),
  UNIQUE KEY `user_info_invitation_code_uindex` (`invitation_code`)
) ENGINE=InnoDB AUTO_INCREMENT=6534 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci ROW_FORMAT=DYNAMIC;*/

type UserInfo struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UUID        string    `gorm:"column:uuid" json:"uuid"`
	UserAddress *string   `gorm:"column:user_address" json:"userAddress"`
	IP          string    `gorm:"column:ip" json:"ip"`
	TwoFA       string    `gorm:"column:two_fa" json:"twoFA"`
	CreateTime  time.Time `gorm:"column:create_time" json:"createTime"`
	UpdateTime  time.Time `gorm:"column:update_time" json:"updateTime"`
}

// 用户信息表
func (UserInfo) TableName() string {
	return "user_info"
}

type MemeVault struct {
	ID           uint64             `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UUID         string             `gorm:"column:uuid" json:"uuid"`
	UserType     string             `gorm:"column:user_type" json:"userType"`
	GroupId      uint64             `gorm:"column:group_id" json:"groupId"`
	ChainIndex   string             `gorm:"column:chain_index" json:"chainIndex"`
	VaultType    int                `gorm:"column:vault_type" json:"vaultType"`
	Status       int                `gorm:"column:status" json:"status"` // 0:正常/未失效  1 注销 2 冻结
	MaxAmount    decimal.Decimal    `gorm:"column:max_amount" json:"maxAmount"`
	MinAmount    decimal.Decimal    `gorm:"column:min_amount" json:"minAmount"`
	StartTime    time.Time          `gorm:"column:start_time" json:"startTime"`
	ExpireTime   time.Time          `gorm:"column:expire_time" json:"expireTime"`
	CreateTime   time.Time          `gorm:"column:create_time" json:"createTime"`
	UpdateTime   time.Time          `gorm:"column:update_time" json:"updateTime"`
	VaultSupport []MemeVaultSupport `gorm:"-" json:"vaultSupport"`
}
type MemeVaultSupport struct {
	ID             uint64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UUID           string          `gorm:"column:uuid" json:"uuid"`
	GroupId        uint64          `gorm:"column:group_id" json:"group_id"`
	WalletID       int64           `gorm:"column:wallet_id" json:"wallet_id"`
	Wallet         string          `gorm:"column:wallet" json:"wallet"`
	FromWallet     string          `gorm:"column:from_wallet" json:"fromWallet"`
	FromWalletID   int64           `gorm:"column:from_wallet_id" json:"fromWalletID"`
	ChainCode      string          `gorm:"column:chain_code" json:"chainCode"`
	VaultType      int             `gorm:"column:vault_type" json:"vaultType"`
	Status         int             `gorm:"column:status" json:"status"` // 0:成功 1:失败
	SupportAddress string          `gorm:"column:support_address" json:"supportAddress"`
	SupportAmount  decimal.Decimal `gorm:"column:support_amount" json:"supportAmount"`
	Price          decimal.Decimal `gorm:"column:price" json:"price"`
	Channel        string          `gorm:"column:channel" json:"channel"`
	CreateTime     time.Time       `gorm:"column:create_time" json:"createTime"`
	UpdateTime     time.Time       `gorm:"column:update_time" json:"updateTime"`
}

func (MemeVault) TableName() string {
	return "meme_vault"
}
func (MemeVaultSupport) TableName() string {
	return "meme_vault_support"
}
