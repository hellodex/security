package store

import (
	"errors"
	"fmt"
	"github.com/hellodex/HelloSecurity/log"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/system"
	"time"
)

var db = system.GetDb()
var mylog = log.GetLogger()

func WalletKeyCheckAndGet(walletKey string) (wk *model.WalletKeys, err error) {
	//校验
	wk = &model.WalletKeys{}
	err = db.Model(&model.WalletKeys{}).Where("wallet_key = ?", walletKey).Take(&wk).Error
	if err != nil {
		return nil, err
	}
	if wk == nil || wk.WalletId <= 0 {
		return nil, errors.New("wallet key not found")
	}
	if wk.ExpireTime > 0 && time.Now().Unix() > wk.ExpireTime {
		return nil, errors.New("wallet key expired")
	}
	return wk, nil
}
func WalletKeySaveBatch(wks []model.WalletKeys) (err error) {
	return db.CreateInBatches(wks, 300).Error
}
func WalletKeyDelByWalletKey(key string) (err error) {
	result := db.Where("wallet_key = ?", key).Delete(&model.WalletKeys{})
	if result.Error != nil {
		return result.Error
	}
	return
}
func WalletKeyDelByUserIdAndChannel(userIdValue string, channelValue string) (err error) {
	result := db.Where("user_id = ? AND channel = ?", userIdValue, channelValue).Delete(&model.WalletKeys{})
	if result.Error != nil {
		return result.Error
	} else {
		fmt.Printf("WalletKeys 删除成功, 删除了 %d 条记录\n", result.RowsAffected)
	}
	return
}

func LimitKeyCheckAndGet(limitKey string) (wk *model.LimitKeys, err error) {
	//校验
	wk = &model.LimitKeys{}
	err = db.Model(&model.LimitKeys{}).Where("limit_key = ?", limitKey).Take(&wk).Error
	if err != nil {
		return nil, err
	}
	if wk == nil || wk.WalletID <= 0 {
		return nil, errors.New("limitKeyNotFound")
	}
	return wk, nil
}
func LimitKeyDelByKey(key string) (err error) {
	result := db.Where("limit_key = ?", key).Delete(&model.LimitKeys{})
	if result.Error != nil {
		return result.Error
	}
	return
}
func LimitKeySave(l model.LimitKeys) (err error) {
	err = db.Model(&model.LimitKeys{}).Save(&l).Error
	return
}

// ========== task_wallet_keys 表操作 ==========

// 校验并获取跟单任务密钥（通过 taskWalletKey 查询）
// 调用链路: AuthSig(channel=="10003") → 本方法
func TaskKeyCheckAndGet(taskWalletKey string) (tk *model.TaskWalletKeys, err error) {
	tk = &model.TaskWalletKeys{}
	err = db.Model(&model.TaskWalletKeys{}).Where("task_wallet_key = ?", taskWalletKey).Take(&tk).Error
	if err != nil {
		return nil, err
	}
	if tk == nil || tk.WalletID <= 0 {
		return nil, errors.New("taskWalletKeyNotFound")
	}
	return tk, nil
}

// 根据 uuid+walletId 查询密钥（用于密钥复用判断）
// 调用链路: TrackTradeCreate/TrackTradeUpdate → 本方法
func TaskWalletKeyGetByUuidAndWallet(uuid int64, walletId uint64) (*model.TaskWalletKeys, error) {
	var tk model.TaskWalletKeys
	err := db.Model(&model.TaskWalletKeys{}).Where("uuid = ? AND wallet_id = ?", uuid, walletId).Take(&tk).Error
	if err != nil {
		return nil, err
	}
	return &tk, nil
}

// 保存密钥
// 调用链路: TrackTradeCreate/TrackTradeUpdate → 本方法
func TaskWalletKeySave(tk model.TaskWalletKeys) error {
	return db.Create(&tk).Error
}

// 删除密钥（按 uuid+walletId，无引用时清理）
// 调用链路: TrackTradeDelete/TrackTradeUpdate → 本方法
func TaskWalletKeyDeleteByUuidAndWallet(uuid int64, walletId uint64) error {
	return db.Where("uuid = ? AND wallet_id = ?", uuid, walletId).Delete(&model.TaskWalletKeys{}).Error
}
