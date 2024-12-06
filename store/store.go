package store

import (
	"errors"
	"fmt"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/system"
	"time"
)

var db = system.GetDb()

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
