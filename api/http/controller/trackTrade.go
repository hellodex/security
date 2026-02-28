package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/codes"
	"github.com/hellodex/HelloSecurity/config"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/store"
)

// ========== 请求结构体 ==========

// 创建跟单任务请求（API → Security）
// 调用链路: API TrackTradeService.createTrackTradeTask() → Security 本接口
type TrackTradeCreateReq struct {
	TaskId        int64           `json:"taskId"`        // 预生成的任务ID
	UUID          int64           `json:"uuid"`          // 用户ID
	WalletKey     string          `json:"walletKey"`     // 钱包密钥（用于身份验证）
	WalletIds     json.RawMessage `json:"walletIds"`     // 钱包ID列表（JSON数组）
	TradeType     int             `json:"tradeType"`     // 交易类型（100=单钱包, 101=多钱包）
	TaskName      string          `json:"taskName"`      // 任务名称
	Status        int             `json:"status"`        // 任务状态（1=运行中）
	WalletAddress []string        `json:"walletAddress"` // 监控地址列表
	Config        json.RawMessage `json:"config"`        // 任务配置
}

// 删除跟单任务请求（API → Security）
// 调用链路: API TrackTradeService.deleteTrackTradeTask() → Security 本接口
type TrackTradeDeleteReq struct {
	TaskId        int64    `json:"taskId"`        // 任务ID
	UUID          int64    `json:"uuid"`          // 用户ID
	WalletAddress []string `json:"walletAddress"` // 监控地址列表（用于通知Task清理addrMap）
}

// 编辑跟单任务请求（API → Security）
// 调用链路: API TrackTradeService.updateTrackTradeTask() → Security 本接口
type TrackTradeUpdateReq struct {
	TaskId           int64           `json:"taskId"`           // 任务ID
	UUID             int64           `json:"uuid"`             // 用户ID
	WalletIds        json.RawMessage `json:"walletIds"`        // 新钱包ID列表
	WalletAddress    []string        `json:"walletAddress"`    // 新监控地址列表
	OldWalletAddress []string        `json:"oldWalletAddress"` // 旧监控地址列表（用于计算addrRemove）
	Config           json.RawMessage `json:"config"`           // 任务配置
	TaskName         string          `json:"taskName"`         // 任务名称
	Status           int             `json:"status"`           // 任务状态
	TradeType        int             `json:"tradeType"`        // 交易类型
}

// 暂停/恢复跟单任务请求（API → Security）
// 调用链路: API TrackTradeService.pauseTrackTradeTask() → Security 本接口
type TrackTradePauseReq struct {
	TaskId int64 `json:"taskId"` // 任务ID
	UUID   int64 `json:"uuid"`  // 用户ID
	Status int   `json:"status"` // 任务状态（0=暂停, 1=运行中）
}

// ========== Handler ==========

// TrackTradeCreate 创建跟单任务
// 流程: 验证walletKey → 对每个walletId生成/复用密钥 → 写task_wallet_ref → HTTP同步Task
// 调用链路: API TrackTradeService.createTrackTradeTask() → Security 本方法 → store层 + httpSyncTask()
func TrackTradeCreate(c *gin.Context) {
	var req TrackTradeCreateReq
	res := common.Response{Timestamp: time.Now().Unix()}
	if err := c.ShouldBindJSON(&req); err != nil {
		mylog.Infof("创建跟单任务参数解析失败, err=%v", err)
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "Invalid request"
		c.JSON(http.StatusOK, res)
		return
	}

	mylog.Infof("创建跟单任务, taskId=%d, uuid=%d, walletIds=%s, addressCount=%d",
		req.TaskId, req.UUID, string(req.WalletIds), len(req.WalletAddress))

	// 1. 验证walletKey
	wk, err := store.WalletKeyCheckAndGet(req.WalletKey)
	if err != nil || wk == nil {
		mylog.Infof("创建跟单任务失败, walletKey验证失败, taskId=%d, err=%v", req.TaskId, err)
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "walletKey验证失败"
		c.JSON(http.StatusOK, res)
		return
	}

	// 2. 解析walletIds
	walletIds := parseWalletIds(req.WalletIds)
	if len(walletIds) == 0 {
		mylog.Infof("创建跟单任务失败, walletIds为空, taskId=%d", req.TaskId)
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "walletIds不能为空"
		c.JSON(http.StatusOK, res)
		return
	}

	// 3. 对每个walletId: 查task_wallet_keys是否已有uuid+walletId的key → 有复用，无则生成
	for _, wid := range walletIds {
		existingKey, _ := store.TaskWalletKeyGetByUuidAndWallet(req.UUID, wid)
		if existingKey == nil {
			// 生成新key（复用common.MyIDStr，与walletKey生成方式一致）
			newKey := common.MyIDStr()
			tk := model.TaskWalletKeys{
				UUID:          req.UUID,
				WalletID:      wid,
				TaskWalletKey: newKey,
			}
			if err := store.TaskWalletKeySave(tk); err != nil {
				mylog.Infof("创建跟单任务失败, 保存密钥失败, taskId=%d, walletId=%d, err=%v", req.TaskId, wid, err)
				res.Code = codes.CODE_ERR_INVALID
				res.Msg = "密钥保存失败"
				c.JSON(http.StatusOK, res)
				return
			}
			mylog.Infof("生成新密钥, taskId=%d, uuid=%d, walletId=%d", req.TaskId, req.UUID, wid)
		} else {
			mylog.Infof("复用已有密钥, taskId=%d, uuid=%d, walletId=%d", req.TaskId, req.UUID, wid)
		}

		// 4. 写task_wallet_ref关联记录
		ref := model.TaskWalletRef{
			TaskID:   req.TaskId,
			WalletID: wid,
			UUID:     req.UUID,
		}
		if err := store.TaskWalletRefSave(ref); err != nil {
			mylog.Infof("创建跟单任务失败, 保存引用失败, taskId=%d, walletId=%d, err=%v", req.TaskId, wid, err)
			res.Code = codes.CODE_ERR_INVALID
			res.Msg = "引用保存失败"
			c.JSON(http.StatusOK, res)
			return
		}
	}

	// 5. HTTP同步通知Task（同步调用，非异步goroutine）
	// addrAdd: 所有地址 → [taskId]
	addrAdd := make(map[string][]int64)
	for _, addr := range req.WalletAddress {
		addrAdd[addr] = []int64{req.TaskId}
	}
	task := map[string]interface{}{
		"id":        req.TaskId,
		"tradeType": req.TradeType,
		"status":    req.Status,
		"config":    req.Config,
	}
	if err := httpSyncTask(req.TaskId, task, addrAdd, nil); err != nil {
		mylog.Infof("创建跟单任务失败, HTTP同步Task失败, taskId=%d, err=%v", req.TaskId, err)
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "通知Task失败"
		c.JSON(http.StatusOK, res)
		return
	}

	mylog.Infof("创建跟单任务成功, taskId=%d, uuid=%d", req.TaskId, req.UUID)
	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	c.JSON(http.StatusOK, res)
}

// TrackTradeDelete 删除跟单任务
// 流程: 删task_wallet_ref → 检查被删walletId引用计数 → 无引用则删key → HTTP通知Task
// 调用链路: API TrackTradeService.deleteTrackTradeTask() → Security 本方法 → store层 + httpDeleteTask()
func TrackTradeDelete(c *gin.Context) {
	var req TrackTradeDeleteReq
	res := common.Response{Timestamp: time.Now().Unix()}
	if err := c.ShouldBindJSON(&req); err != nil {
		mylog.Infof("删除跟单任务参数解析失败, err=%v", err)
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "Invalid request"
		c.JSON(http.StatusOK, res)
		return
	}

	mylog.Infof("删除跟单任务, taskId=%d, uuid=%d", req.TaskId, req.UUID)

	// 1. 删除task_wallet_ref中该taskId的所有关联记录，返回被删记录用于后续key清理
	deletedRefs, err := store.TaskWalletRefDeleteByTaskId(req.TaskId)
	if err != nil {
		mylog.Infof("删除跟单任务失败, 删除引用失败, taskId=%d, err=%v", req.TaskId, err)
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "删除引用失败"
		c.JSON(http.StatusOK, res)
		return
	}

	// 2. 检查被删walletId是否还有其他任务引用，无引用则删除key
	for _, ref := range deletedRefs {
		count, err := store.TaskWalletRefCountByUuidAndWallet(ref.UUID, ref.WalletID)
		if err != nil {
			mylog.Infof("删除跟单任务, 查询引用计数失败, uuid=%d, walletId=%d, err=%v", ref.UUID, ref.WalletID, err)
			continue
		}
		if count == 0 {
			if err := store.TaskWalletKeyDeleteByUuidAndWallet(ref.UUID, ref.WalletID); err != nil {
				mylog.Infof("删除跟单任务, 清理无引用密钥失败, uuid=%d, walletId=%d, err=%v", ref.UUID, ref.WalletID, err)
			} else {
				mylog.Infof("清理无引用密钥, uuid=%d, walletId=%d", ref.UUID, ref.WalletID)
			}
		} else {
			mylog.Infof("密钥仍被引用, uuid=%d, walletId=%d, refCount=%d", ref.UUID, ref.WalletID, count)
		}
	}

	// 3. HTTP通知Task删除（同步调用）
	if err := httpDeleteTask(req.TaskId, req.WalletAddress); err != nil {
		mylog.Infof("删除跟单任务失败, HTTP通知Task失败, taskId=%d, err=%v", req.TaskId, err)
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "通知Task失败"
		c.JSON(http.StatusOK, res)
		return
	}

	mylog.Infof("删除跟单任务成功, taskId=%d", req.TaskId)
	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	c.JSON(http.StatusOK, res)
}

// TrackTradeUpdate 编辑跟单任务
// 流程: 计算walletIds diff → 新增生成/复用key → 移除检查引用清理key → 更新ref → HTTP转发Task
// 调用链路: API TrackTradeService.updateTrackTradeTask() → Security 本方法 → store层 + httpSyncTask()
func TrackTradeUpdate(c *gin.Context) {
	var req TrackTradeUpdateReq
	res := common.Response{Timestamp: time.Now().Unix()}
	if err := c.ShouldBindJSON(&req); err != nil {
		mylog.Infof("编辑跟单任务参数解析失败, err=%v", err)
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "Invalid request"
		c.JSON(http.StatusOK, res)
		return
	}

	mylog.Infof("编辑跟单任务, taskId=%d, uuid=%d, newWalletIds=%s", req.TaskId, req.UUID, string(req.WalletIds))

	// 1. 获取旧的引用记录（用于计算walletIds diff）
	oldRefs, err := store.TaskWalletRefGetByTaskId(req.TaskId)
	if err != nil {
		mylog.Infof("编辑跟单任务失败, 查询旧引用失败, taskId=%d, err=%v", req.TaskId, err)
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "查询旧引用失败"
		c.JSON(http.StatusOK, res)
		return
	}
	oldWalletIdSet := make(map[uint64]bool)
	for _, ref := range oldRefs {
		oldWalletIdSet[ref.WalletID] = true
	}

	// 2. 解析新的walletIds
	newWalletIds := parseWalletIds(req.WalletIds)
	newWalletIdSet := make(map[uint64]bool)
	for _, wid := range newWalletIds {
		newWalletIdSet[wid] = true
	}

	// 3. 处理新增的walletIds（在新列表中但不在旧列表中）
	for _, wid := range newWalletIds {
		if oldWalletIdSet[wid] {
			continue
		}
		// 新增walletId: 生成/复用key
		existingKey, _ := store.TaskWalletKeyGetByUuidAndWallet(req.UUID, wid)
		if existingKey == nil {
			newKey := common.MyIDStr()
			tk := model.TaskWalletKeys{
				UUID:          req.UUID,
				WalletID:      wid,
				TaskWalletKey: newKey,
			}
			if err := store.TaskWalletKeySave(tk); err != nil {
				mylog.Infof("编辑跟单任务, 保存新密钥失败, taskId=%d, walletId=%d, err=%v", req.TaskId, wid, err)
				continue
			}
			mylog.Infof("编辑任务生成新密钥, taskId=%d, uuid=%d, walletId=%d", req.TaskId, req.UUID, wid)
		} else {
			mylog.Infof("编辑任务复用已有密钥, taskId=%d, uuid=%d, walletId=%d", req.TaskId, req.UUID, wid)
		}
		// 写入新的ref
		ref := model.TaskWalletRef{TaskID: req.TaskId, WalletID: wid, UUID: req.UUID}
		if err := store.TaskWalletRefSave(ref); err != nil {
			mylog.Infof("编辑跟单任务, 保存新引用失败, taskId=%d, walletId=%d, err=%v", req.TaskId, wid, err)
		}
	}

	// 4. 处理移除的walletIds（在旧列表中但不在新列表中）
	for _, ref := range oldRefs {
		if newWalletIdSet[ref.WalletID] {
			continue
		}
		// 删除ref
		if err := store.TaskWalletRefDeleteByTaskAndWallet(req.TaskId, ref.WalletID); err != nil {
			mylog.Infof("编辑跟单任务, 删除旧引用失败, taskId=%d, walletId=%d, err=%v", req.TaskId, ref.WalletID, err)
			continue
		}
		// 检查该uuid+walletId是否还有其他任务引用
		count, err := store.TaskWalletRefCountByUuidAndWallet(ref.UUID, ref.WalletID)
		if err != nil {
			mylog.Infof("编辑跟单任务, 查询引用计数失败, uuid=%d, walletId=%d, err=%v", ref.UUID, ref.WalletID, err)
			continue
		}
		if count == 0 {
			if err := store.TaskWalletKeyDeleteByUuidAndWallet(ref.UUID, ref.WalletID); err != nil {
				mylog.Infof("编辑跟单任务, 清理无引用密钥失败, uuid=%d, walletId=%d, err=%v", ref.UUID, ref.WalletID, err)
			} else {
				mylog.Infof("编辑任务清理无引用密钥, uuid=%d, walletId=%d", ref.UUID, ref.WalletID)
			}
		}
	}

	// 5. 计算地址diff（addrAdd/addrRemove）
	addrAdd := make(map[string][]int64)
	var addrRemove []string
	newAddrSet := make(map[string]bool)
	for _, addr := range req.WalletAddress {
		newAddrSet[addr] = true
	}
	oldAddrSet := make(map[string]bool)
	for _, addr := range req.OldWalletAddress {
		oldAddrSet[addr] = true
	}
	// addrAdd: 新增的地址
	for _, addr := range req.WalletAddress {
		if !oldAddrSet[addr] {
			addrAdd[addr] = []int64{req.TaskId}
		}
	}
	// addrRemove: 移除的地址
	for _, addr := range req.OldWalletAddress {
		if !newAddrSet[addr] {
			addrRemove = append(addrRemove, addr)
		}
	}

	// 6. HTTP同步Task
	task := map[string]interface{}{
		"id":        req.TaskId,
		"tradeType": req.TradeType,
		"status":    req.Status,
		"config":    req.Config,
	}
	if err := httpSyncTask(req.TaskId, task, addrAdd, addrRemove); err != nil {
		mylog.Infof("编辑跟单任务失败, HTTP同步Task失败, taskId=%d, err=%v", req.TaskId, err)
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "通知Task失败"
		c.JSON(http.StatusOK, res)
		return
	}

	mylog.Infof("编辑跟单任务成功, taskId=%d", req.TaskId)
	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	c.JSON(http.StatusOK, res)
}

// TrackTradePause 暂停或恢复跟单任务
// 流程: 不涉及密钥操作，直接HTTP转发Task（syncTask，仅status变更）
// 调用链路: API TrackTradeService.pauseTrackTradeTask() → Security 本方法 → httpSyncTask()
func TrackTradePause(c *gin.Context) {
	var req TrackTradePauseReq
	res := common.Response{Timestamp: time.Now().Unix()}
	if err := c.ShouldBindJSON(&req); err != nil {
		mylog.Infof("暂停/恢复跟单任务参数解析失败, err=%v", err)
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "Invalid request"
		c.JSON(http.StatusOK, res)
		return
	}

	mylog.Infof("暂停/恢复跟单任务, taskId=%d, uuid=%d, status=%d", req.TaskId, req.UUID, req.Status)

	// 直接转发Task（syncTask，仅status变更，addrAdd和addrRemove为空）
	task := map[string]interface{}{
		"id":     req.TaskId,
		"status": req.Status,
	}
	if err := httpSyncTask(req.TaskId, task, nil, nil); err != nil {
		mylog.Infof("暂停/恢复跟单任务失败, HTTP同步Task失败, taskId=%d, err=%v", req.TaskId, err)
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "通知Task失败"
		c.JSON(http.StatusOK, res)
		return
	}

	mylog.Infof("暂停/恢复跟单任务成功, taskId=%d, status=%d", req.TaskId, req.Status)
	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	c.JSON(http.StatusOK, res)
}

// ========== HTTP通信（同步调用Task） ==========

// httpSyncTask 同步调用Task工程的syncTask接口（创建/编辑/暂停/恢复）
// 调用链路: TrackTradeCreate/Update/Pause → 本方法 → Task POST /internal/trackTrade/syncTask
func httpSyncTask(taskId int64, task map[string]interface{}, addrAdd map[string][]int64, addrRemove []string) error {
	body := map[string]interface{}{
		"taskId":     taskId,
		"task":       task,
		"addrAdd":    addrAdd,
		"addrRemove": addrRemove,
	}
	return httpPostToTask("/internal/trackTrade/syncTask", body)
}

// httpDeleteTask 同步调用Task工程的deleteTask接口
// 调用链路: TrackTradeDelete → 本方法 → Task POST /internal/trackTrade/deleteTask
func httpDeleteTask(taskId int64, addrRemove []string) error {
	body := map[string]interface{}{
		"taskId":     taskId,
		"addrRemove": addrRemove,
	}
	return httpPostToTask("/internal/trackTrade/deleteTask", body)
}

// httpPostToTask 通用HTTP POST请求到Task工程（同步调用，非异步goroutine）
// 调用链路: httpSyncTask/httpDeleteTask → 本方法
func httpPostToTask(path string, body map[string]interface{}) error {
	cfg := config.GetConfig()
	if cfg.TaskServiceUrl == "" {
		mylog.Infof("Task工程地址未配置, 跳过HTTP通知, path=%s", path)
		return nil
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("JSON序列化失败: %v", err)
	}

	url := cfg.TaskServiceUrl + path
	mylog.Infof("HTTP请求Task工程, url=%s, body=%s", url, string(jsonData))

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Task返回异常, statusCode=%d, body=%s", resp.StatusCode, string(respBody))
	}

	mylog.Infof("HTTP请求Task工程成功, url=%s, respStatus=%d", url, resp.StatusCode)
	return nil
}

// ========== 工具方法 ==========

// parseWalletIds 解析walletIds JSON数组为uint64切片
// walletIds可能是 [12345, 67890] 或 ["12345", "67890"] 格式（Java JSONArray序列化差异）
// 调用链路: TrackTradeCreate/TrackTradeUpdate → 本方法
func parseWalletIds(raw json.RawMessage) []uint64 {
	if len(raw) == 0 {
		return nil
	}

	// 尝试解析为float64数组（JSON number默认解析为float64）
	var floatIds []float64
	if err := json.Unmarshal(raw, &floatIds); err == nil {
		result := make([]uint64, 0, len(floatIds))
		for _, f := range floatIds {
			result = append(result, uint64(f))
		}
		return result
	}

	// 尝试解析为字符串数组（API可能发送字符串格式的数字）
	var strIds []string
	if err := json.Unmarshal(raw, &strIds); err == nil {
		result := make([]uint64, 0, len(strIds))
		for _, s := range strIds {
			var id uint64
			if _, err := fmt.Sscanf(s, "%d", &id); err == nil {
				result = append(result, id)
			}
		}
		return result
	}

	return nil
}
