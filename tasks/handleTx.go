package tasks

import (
	"context"
	"github.com/bruceshao/lockfree"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/config"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/system"
	"strconv"
	"time"
)

var (
	lf         *lockfree.Lockfree[model.MemeVaultSupport]
	client     *rpc.Client
	maxVersion = uint64(0)
)

func init() {
	// 创建事件处理器
	handler := &eventHandler[model.MemeVaultSupport]{
		signal: make(chan struct{}),
		now:    time.Now(),
	}
	lf = lockfree.NewLockfree[model.MemeVaultSupport](
		1024*1024,
		handler,
		lockfree.NewSleepBlockStrategy(time.Millisecond),
	)
	client = rpc.New(config.GetRpcConfig("SOLANA").GetRpc()[0])
}
func HandleTx(tx model.MemeVaultSupport) {
	lf.Producer().Write(tx)
}

func (h *eventHandler[T]) OnEvent(v model.MemeVaultSupport) {
	lock := common.GetLock("txStatus" + strconv.FormatUint(v.ID, 10))
	if lock.Lock.TryLock() {
		defer lock.Lock.Unlock()
	} else {
		return
	}
	// todo 业务频繁了，这里需要优化
	if h.lastHandler.Before(time.Now().Add(-time.Minute * 5)) {
		h.processed = make(map[uint64]model.MemeVaultSupport)
	}
	h.lastHandler = time.Now()
	_, processed := h.processed[v.ID]
	if processed {
		return
	}

	out, err := client.GetTransaction(
		context.TODO(),
		solana.MustSignatureFromBase58(v.Tx),
		&rpc.GetTransactionOpts{
			Encoding:                       solana.EncodingBase64,
			MaxSupportedTransactionVersion: &maxVersion,
		},
	)
	if err != nil || out == nil || out.Meta == nil {
		return
	}
	status := 205
	if v.Status == 201 {
		status = 202
	}
	if v.CreateTime.Before(time.Now().Add(-time.Minute * 15)) {
		status = 204
	}
	if out.Meta.Err == nil {
		status = 200
	} else {
		status = 205
	}
	db := system.GetDb()
	if db != nil {
		task := func() {
			var inDB model.MemeVaultSupport
			err = db.Model(&model.MemeVaultSupport{}).Where("id = ?", v.ID).First(&inDB).Error
			ups := map[string]interface{}{
				"update_time": time.Now(),
			}
			if inDB.Status != status {
				ups["status"] = status
			}
			if err = db.Model(&model.MemeVaultSupport{}).Where("id = ?", v.ID).Updates(ups).Error; err == nil {
				v.Status = status
				h.processed[v.ID] = v
			}
		}
		task()
	}
}
func (h *eventHandler[T]) wait() {
	<-h.signal
}

type eventHandler[T any] struct {
	signal      chan struct{}
	processed   map[uint64]model.MemeVaultSupport
	lastHandler time.Time
	now         time.Time
}
