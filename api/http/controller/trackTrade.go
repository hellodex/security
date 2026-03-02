package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/codes"
	"github.com/hellodex/HelloSecurity/config"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/store"
)

// ========== 请求结构体 ==========

// TaskWalletIdItem 任务钱包ID项（每个选中钱包的walletId + walletKey）
// 敏感数据：walletKey 只在 Security 层验证后使用，不存储到 API/Task
type TaskWalletIdItem struct {
	WalletId  interface{} `json:"walletId"`  // 钱包ID（兼容数字和字符串格式）
	WalletKey string      `json:"walletKey"` // 钱包密钥（用于身份验证）
}

// 创建跟单任务请求（API → Security）
// 调用链路: API TrackTradeService.createTrackTradeTask() → Security 本接口
// 重构: taskWalletIds 替代原来的 walletKey+walletIds，config 包含所有业务字段
type TrackTradeCreateReq struct {
	TaskId        int64              `json:"taskId"`        // 预生成的任务ID
	UUID          int64              `json:"uuid"`          // 用户ID
	TaskWalletIds []TaskWalletIdItem `json:"taskWalletIds"` // [{walletId, walletKey}] 每个钱包独立验证
	Config        json.RawMessage    `json:"config"`        // 完整配置（含tradeType/taskName/status/trackWalletAddress等）
}

// 删除跟单任务请求（API → Security）
// 调用链路: API TrackTradeService.deleteTrackTradeTask() → Security 本接口
type TrackTradeDeleteReq struct {
	TaskId             int64    `json:"taskId"`             // 任务ID
	UUID               int64    `json:"uuid"`               // 用户ID
	TrackWalletAddress []string `json:"trackWalletAddress"` // 监控地址列表（通知Task清理addrMap）
}

// 编辑跟单任务请求（API → Security）
// 调用链路: API TrackTradeService.updateTrackTradeTask() → Security 本接口
// 重构: taskWalletIds 替代原来的 walletIds，config 包含所有业务字段
type TrackTradeUpdateReq struct {
	TaskId                int64              `json:"taskId"`                // 任务ID
	UUID                  int64              `json:"uuid"`                  // 用户ID
	TaskWalletIds         []TaskWalletIdItem `json:"taskWalletIds"`         // [{walletId, walletKey}] 可选（钱包变更时传入）
	Config                json.RawMessage    `json:"config"`                // 新的完整配置
	OldTrackWalletAddress []string           `json:"oldTrackWalletAddress"` // 旧的监控地址（用于计算addrRemove）
}

// 暂停/恢复跟单任务请求（API → Security）
// 调用链路: API TrackTradeService.pauseTrackTradeTask() → Security 本接口
type TrackTradePauseReq struct {
	TaskId int64 `json:"taskId"` // 任务ID
	UUID   int64 `json:"uuid"`   // 用户ID
	Status int   `json:"status"` // 任务状态（0=暂停, 1=运行中）
}

// ========== Handler ==========

// TrackTradeCreate 创建跟单任务
// 流程: 遍历taskWalletIds逐个验证walletKey → 生成/复用密钥 → 写task_wallet_ref → HTTP同步Task
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

	mylog.Infof("创建跟单任务, taskId=%d, uuid=%d, walletCount=%d",
		req.TaskId, req.UUID, len(req.TaskWalletIds))

	// 1. 校验taskWalletIds不为空
	if len(req.TaskWalletIds) == 0 {
		mylog.Infof("创建跟单任务失败, taskWalletIds为空, taskId=%d", req.TaskId)
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "taskWalletIds不能为空"
		c.JSON(http.StatusOK, res)
		return
	}

	// 2. 遍历taskWalletIds，逐个验证walletKey + 生成/复用密钥 + 写ref
	for _, item := range req.TaskWalletIds {
		wid := parseWalletIdToUint64(item.WalletId)
		if wid == 0 {
			mylog.Infof("创建跟单任务失败, walletId解析失败, taskId=%d, walletId=%v", req.TaskId, item.WalletId)
			res.Code = codes.CODE_ERR_INVALID
			res.Msg = "walletId格式错误"
			c.JSON(http.StatusOK, res)
			return
		}

		// 验证walletKey（确认用户持有该钱包）
		wk, err := store.WalletKeyCheckAndGet(item.WalletKey)
		if err != nil || wk == nil {
			mylog.Infof("创建跟单任务失败, walletKey验证失败, taskId=%d, walletId=%d, err=%v", req.TaskId, wid, err)
			res.Code = codes.CODE_ERR_INVALID
			res.Msg = "walletKey验证失败"
			c.JSON(http.StatusOK, res)
			return
		}

		// 安全校验：walletKey对应的walletId必须与请求中的walletId一致
		if wk.WalletId != wid {
			mylog.Infof("创建跟单任务失败, walletId不匹配, taskId=%d, reqWalletId=%d, keyWalletId=%d", req.TaskId, wid, wk.WalletId)
			res.Code = codes.CODE_ERR_INVALID
			res.Msg = "walletId与walletKey不匹配"
			c.JSON(http.StatusOK, res)
			return
		}

		// 生成/复用 taskWalletKey
		existingKey, _ := store.TaskWalletKeyGetByUuidAndWallet(req.UUID, wid)
		if existingKey == nil {
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

		// 写task_wallet_ref关联记录
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

	// 3. 从config中提取trackWalletAddress，构建addrAdd
	trackAddrs := extractTrackWalletAddress(req.Config)
	addrAdd := make(map[string][]int64)
	for _, addr := range trackAddrs {
		addrAdd[addr] = []int64{req.TaskId}
	}

	// 4. HTTP同步通知Task（task结构: {id, uuid, config}）
	task := map[string]interface{}{
		"id":     req.TaskId,
		"uuid":   req.UUID,
		"config": req.Config,
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

	// 3. HTTP通知Task删除（trackWalletAddress用于Task清理addrMap）
	if err := httpDeleteTask(req.TaskId, req.TrackWalletAddress); err != nil {
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
// 流程: 处理taskWalletIds变更(如有) → 计算地址diff → HTTP转发Task
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

	mylog.Infof("编辑跟单任务, taskId=%d, uuid=%d, walletCount=%d", req.TaskId, req.UUID, len(req.TaskWalletIds))

	// 1. 如果有taskWalletIds，处理钱包变更（验证 + 密钥管理 + ref更新）
	if len(req.TaskWalletIds) > 0 {
		// 获取旧的引用记录（用于计算walletIds diff）
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

		// 解析新的walletIds
		newWalletIdSet := make(map[uint64]bool)
		for _, item := range req.TaskWalletIds {
			wid := parseWalletIdToUint64(item.WalletId)
			if wid == 0 {
				continue
			}
			newWalletIdSet[wid] = true
		}

		// 处理新增的walletIds
		for _, item := range req.TaskWalletIds {
			wid := parseWalletIdToUint64(item.WalletId)
			if wid == 0 || oldWalletIdSet[wid] {
				continue
			}

			// 验证walletKey
			wk, err := store.WalletKeyCheckAndGet(item.WalletKey)
			if err != nil || wk == nil {
				mylog.Infof("编辑跟单任务, walletKey验证失败, taskId=%d, walletId=%d, err=%v", req.TaskId, wid, err)
				continue
			}
			if wk.WalletId != wid {
				mylog.Infof("编辑跟单任务, walletId不匹配, taskId=%d, reqWalletId=%d, keyWalletId=%d", req.TaskId, wid, wk.WalletId)
				continue
			}

			// 生成/复用key
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

		// 处理移除的walletIds（在旧列表中但不在新列表中）
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
	}

	// 2. 计算地址diff（从config提取新的trackWalletAddress，与旧的OldTrackWalletAddress对比）
	newTrackAddrs := extractTrackWalletAddress(req.Config)
	addrAdd := make(map[string][]int64)
	var addrRemove []string
	newAddrSet := make(map[string]bool)
	for _, addr := range newTrackAddrs {
		newAddrSet[addr] = true
	}
	oldAddrSet := make(map[string]bool)
	for _, addr := range req.OldTrackWalletAddress {
		oldAddrSet[addr] = true
	}
	// addrAdd: 新增的地址
	for _, addr := range newTrackAddrs {
		if !oldAddrSet[addr] {
			addrAdd[addr] = []int64{req.TaskId}
		}
	}
	// addrRemove: 移除的地址
	for _, addr := range req.OldTrackWalletAddress {
		if !newAddrSet[addr] {
			addrRemove = append(addrRemove, addr)
		}
	}

	// 3. HTTP同步Task（task结构: {id, uuid, config}）
	task := map[string]interface{}{
		"id":     req.TaskId,
		"uuid":   req.UUID,
		"config": req.Config,
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

// parseWalletIdToUint64 将walletId（可能是数字或字符串格式）解析为uint64
// 调用链路: TrackTradeCreate/TrackTradeUpdate → 本方法
func parseWalletIdToUint64(v interface{}) uint64 {
	switch val := v.(type) {
	case float64:
		return uint64(val)
	case json.Number:
		n, _ := val.Int64()
		return uint64(n)
	case string:
		id, _ := strconv.ParseUint(val, 10, 64)
		return id
	}
	return 0
}

// extractTrackWalletAddress 从config JSON中提取trackWalletAddress字段
// 调用链路: TrackTradeCreate/TrackTradeUpdate → 本方法
func extractTrackWalletAddress(configRaw json.RawMessage) []string {
	if len(configRaw) == 0 {
		return nil
	}
	var configMap map[string]interface{}
	if err := json.Unmarshal(configRaw, &configMap); err != nil {
		return nil
	}
	addrs, ok := configMap["trackWalletAddress"]
	if !ok {
		return nil
	}
	arr, ok := addrs.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if m, ok := v.(map[string]interface{}); ok {
			if addr, ok := m["walletAddress"].(string); ok && addr != "" {
				result = append(result, addr)
			}
		}
	}
	return result
}
