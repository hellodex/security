package swapData

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/shopspring/decimal"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

// JupSwapReq 结构体定义请求参数

func GetSwapDataByJupApi(retries int, s map[string]interface{}, params *common.LimitOrderParam) (map[string]interface{}, map[string]interface{}, error) {
	api, response, err := getSwapDate(params)
	if response != nil {
		// 检查响应中的 code
		if code, ok := response["code"].(float64); ok && int(code) == 200 {
			if data, ok := response["data"].(map[string]interface{}); ok {
				if swapRes, ok := data["swapRes"].(map[string]interface{}); ok {
					if swapTransaction, ok := swapRes["swapTransaction"].(map[string]interface{}); ok {
						response["singData"] = swapTransaction
					}
				}
			}
		}
	}
	swapmap := make(map[string]interface{})
	swapmap["req"] = api
	swapmap["res"] = response
	s["swapData"+strconv.Itoa(retries)] = swapmap
	return s, response, err
}

// getSwapDate 发送 POST 请求并返回响应数据
func getSwapDate(req *common.LimitOrderParam) (common.LimitOrderParam, map[string]interface{}, error) {
	// 构建请求参数
	// Build query parameters
	slippage, err := decimal.NewFromString(req.Slippage)
	if err != nil {
		slippage = decimal.NewFromFloat(0.05) // Default slippage to 1%
	}
	paramMap := make(map[string]interface{})
	paramMap["amount"] = req.Amount.String()
	if req.FromTokenAddress == "11111111111111111111111111111111" {
		req.FromTokenAddress = "So11111111111111111111111111111111111111112"
	}
	if req.ToTokenAddress == "11111111111111111111111111111111" {
		req.ToTokenAddress = "So11111111111111111111111111111111111111112"
	}
	paramMap["fromTokenAddress"] = req.FromTokenAddress
	paramMap["toTokenAddress"] = req.ToTokenAddress
	paramMap["slippage"] = slippage.Mul(decimal.NewFromFloat(10000)).BigInt().Int64()
	paramMap["userPublicKey"] = req.UserWalletAddress
	if req.FeeAccount != "" {
		paramMap["feeAccount"] = req.FeeAccount
		paramMap["platformFeebps"] = 100
	}
	if req.DynamicSlippage {
		paramMap["dynamicSlippage"] = true
	}
	if req.DynamicComputeUnitLimit {
		paramMap["dynamicComputeUnitLimit"] = true
	}

	// 将请求参数转换为 JSON
	jsonData, err := json.Marshal(paramMap)
	if err != nil {
		return *req, make(map[string]interface{}), fmt.Errorf("failed to marshal request parameters: %v", err)
	}

	// 构建请求头
	headerMap := map[string]string{
		"content-type": "application/json",
		"XAUTH":        cfg.JupKey,
	}

	// 创建 HTTP 客户端
	client := &http.Client{}

	// 尝试最多 3 次发送 POST 请求
	var result string
	for i := 0; i < 3; i++ {
		// 创建 POST 请求
		httpReq, err := http.NewRequest("POST", cfg.JupKeyHost+"/swap", bytes.NewBuffer(jsonData))
		if err != nil {
			log.Printf("Attempt %d failed to create request: %v", i+1, err)
			continue
		}

		// 设置请求头
		for key, value := range headerMap {
			httpReq.Header.Add(key, value)
		}

		// 发送请求
		resp, err := client.Do(httpReq)
		if err != nil {
			log.Printf("Attempt %d failed to send request: %v", i+1, err)
			continue
		}
		defer resp.Body.Close()

		// 读取响应内容
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Attempt %d failed to read response: %v", i+1, err)
			continue
		}
		result = string(body)
		if result != "{}" {
			break
		}
	}

	// 检查是否获取到有效响应
	if result == "" || result == "{}" {
		return *req, make(map[string]interface{}), fmt.Errorf("failed to get valid response after 3 attempts")
	}

	// 解析 JSON 响应
	var res map[string]interface{}
	err = json.Unmarshal([]byte(result), &res)
	if err != nil {
		return *req, make(map[string]interface{}), fmt.Errorf("failed to parse JSON response: %v", err)
	}

	return *req, res, nil
}

// getSwapDate 发送
