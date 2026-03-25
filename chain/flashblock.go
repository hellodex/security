package chain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/hellodex/HelloSecurity/api/common"
)

// flashBlockClient 复用 HTTP 长连接
var flashBlockClient = &http.Client{
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     60 * time.Second,
	},
	Timeout: 15 * time.Second,
}

// SendTransactionFlashBlock 通过 FlashBlock JSON-RPC 提交交易，返回 tx hash
// 调用链路: HandleMessage → 本方法（当 conf.TxChannel.Type == 1 时）
func SendTransactionFlashBlock(ctx context.Context, tx *solana.Transaction, cfg *common.TxChannelConfig) (string, error) {
	txBase64, err := tx.ToBase64()
	if err != nil {
		return "", fmt.Errorf("FlashBlock 编码交易失败: %v", err)
	}

	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "sendTransaction",
		"params": []interface{}{
			txBase64,
			map[string]string{"encoding": "base64"},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("FlashBlock 序列化请求失败: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", cfg.Url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("FlashBlock 创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", cfg.ApiKey)

	startMs := time.Now().UnixMilli()
	resp, err := flashBlockClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("FlashBlock 发送请求失败: %v, %dms", err, time.Now().UnixMilli()-startMs)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("FlashBlock 读取响应失败: %v", err)
	}

	var rpcResp struct {
		Result interface{}  `json:"result"`
		Error  *interface{} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return "", fmt.Errorf("FlashBlock 解析响应失败: %v, body=%s", err, string(body))
	}
	if rpcResp.Error != nil {
		mylog.Infof("FlashBlock 通道提交-失败-RPC 错误, 耗时：%dms, 传递参数：%s, 返回参数：%s", time.Now().UnixMilli()-startMs, string(jsonData), string(body))
		return "", fmt.Errorf("FlashBlock RPC 错误: %+v", rpcResp.Error)
	}

	txHash, ok := rpcResp.Result.(string)
	if !ok || len(txHash) == 0 {
		mylog.Infof("FlashBlock 通道提交-失败-返回签名为空, 耗时：%dms, 传递参数：%s, 返回参数：%s", time.Now().UnixMilli()-startMs, string(jsonData), string(body))
		return "", fmt.Errorf("FlashBlock 返回签名为空, result=%+v", rpcResp.Result)
	}

	mylog.Infof("FlashBlock 通道提交-成功, 耗时：%dms, 传递参数：%s, 返回参数：%s", time.Now().UnixMilli()-startMs, string(jsonData), string(body))
	return txHash, nil
}
