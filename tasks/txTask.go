package tasks

import (
	"github.com/go-co-op/gocron"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/system"
	"strconv"
	"strings"
	"time"
)

var scheduler *gocron.Scheduler

func init() {
	go func() {
		scheduler = gocron.NewScheduler(time.Local)
		scheduler.Every(3).Second().Do(txStatusTask)
		scheduler.StartAsync()
	}()

}
func txStatusTask() {
	db := system.GetDb()
	if db != nil {
		var vaultSp []model.MemeVaultSupport

		_ = db.Model(&model.MemeVaultSupport{}).Where("create_time >= ? AND status != ? AND status != ? AND tx IS NOT NULL  ",
			time.Now().Add(-time.Minute*30), 200, 205).Find(&vaultSp).Error

		if len(vaultSp) > 0 {
			for _, v := range vaultSp {
				if v.Status < 205 && v.Status > 200 && v.Tx != "" && !strings.HasPrefix(v.Tx, "1111111111111111") {
					func() {
						lock := common.GetLock("txStatus" + strconv.FormatUint(v.ID, 10))
						if lock.Lock.TryLock() {
							defer lock.Lock.Unlock()
							HandleTx(v)
						}
					}()

				}
			}
		}
	}
}
