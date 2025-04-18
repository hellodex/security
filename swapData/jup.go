package swapData

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hellodex/HelloSecurity/api/common"
	mylog "github.com/hellodex/HelloSecurity/log"
	"github.com/hellodex/HelloSecurity/wallet"
	"github.com/shopspring/decimal"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"strings"
)

// JupSwapReq 结构体定义请求参数

func GetSwapDataByJupApi(retries int, s map[string]interface{}, params *common.LimitOrderParam) (map[string]interface{}, map[string]interface{}, error) {

	api, response, err := getSwapDate(params)

	if response != nil {
		// 检查响应中的 code
		if code, ok := response["code"].(float64); ok && int(code) == 200 {
			if !params.IsBuy && params.IsMemeVaultWalletTrade {

			}
			if data, ok := response["data"].(map[string]interface{}); ok {
				if swapRes, ok := data["swapRes"].(map[string]interface{}); ok {
					if swapTransaction, ok := swapRes["swapTransaction"].(string); ok {
						response["singData"] = swapTransaction
					}
				}
				if swapReq, ok := data["swapReq"].(map[string]interface{}); ok {
					if quoteResponse, ok := swapReq["quoteResponse"].(map[string]interface{}); ok {
						outAmountI, ex := quoteResponse["outAmount"]
						outputMintI, ex1 := quoteResponse["outputMint"]
						mylog.Infof("outAmount: %v, outputMint: %v,tip: %d", outAmountI, outputMintI, params.JitoTipLamports)
						if params.IsMemeVaultWalletTrade && ex && ex1 && params.JitoTipLamports.Sign() > 0 &&
							(strings.HasPrefix(outputMintI.(string), "So1111111111111") ||
								strings.HasPrefix(outputMintI.(string), "111111111111111")) {
							userReceive, vaultTip, memeVaultInfo := memeVaultTip(outAmountI.(string), outputMintI.(string), params.Amount, params.FromTokenDecimals, params.ToTokenDecimals,
								params.UserWalletAddress, params.FromTokenAddress, params.TotalVolumeBuy, params.RealizedProfit, params.AvgPrice)
							if vaultTip.Sign() > 0 {
								response["vaultTip"] = vaultTip
							}
							response["memeVaultInfo"] = memeVaultInfo
							response["userReceive"] = userReceive

						}

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

func memeVaultTip(outAmount string, outputMint string, fromAmount *big.Int, fromTokenDecimals int64, toTokenDecimals int64, userWalletAddress string, fromTokenAddress string, totalVolumeBuy decimal.Decimal,
	realizedProfit decimal.Decimal, avgPrice decimal.Decimal) (decimal.Decimal, *big.Int, map[string]interface{}) {

	//价值币 价格
	priceStr := wallet.QuotePrice("SOLANA", outputMint)
	price, _ := decimal.NewFromString(priceStr)
	amount, _ := decimal.NewFromString(outAmount)
	// 价值币数量
	amount = amount.Div(decimal.NewFromInt(10).Pow(decimal.NewFromInt(toTokenDecimals)))
	// 价值币总价值
	receiveAll := price.Mul(amount)
	fAmount := decimal.NewFromBigInt(fromAmount, 0)
	// 卖出meme数量
	fAmount = fAmount.Div(decimal.NewFromInt(10).Pow(decimal.NewFromInt(fromTokenDecimals)))
	costVolume := avgPrice.Mul(fAmount)
	memeVaultInfo := make(map[string]interface{})
	memeVaultInfo["用户钱包"] = userWalletAddress
	memeVaultInfo["代币地址"] = fromTokenAddress
	memeVaultInfo["meme数量"] = fAmount.String()
	memeVaultInfo["平均买入价格"] = avgPrice.String()
	memeVaultInfo["sol价格"] = priceStr
	memeVaultInfo["总成本买入金额"] = totalVolumeBuy.String()
	memeVaultInfo["累计卖出仓位已获得金额"] = realizedProfit.String()
	memeVaultInfo["本次交易成本金额"] = costVolume.String()
	memeVaultInfo["本次交易获得SOL"] = amount.String()
	memeVaultInfo["本次交易获得金额"] = receiveAll.String()
	memeVaultInfo["本次交易盈利SOL"] = amount.String()
	memeVaultInfo["本次交易盈利金额"] = receiveAll.String()
	memeVaultInfo["用户本次收到的SOL"] = amount.String()
	memeVaultInfo["用户本次收到金额"] = receiveAll.String()
	memeVaultInfo["冲狗基金本次收到SOL"] = "0"
	memeVaultInfo["冲狗基金本次收到金额"] = "0"
	memeVaultInfo["用户加本次累计卖出仓位总获得金额"] = realizedProfit.Add(receiveAll).String()
	memeVaultInfo["冲狗基金回本"] = "未回本"
	vaultTip := big.NewInt(0)
	userReceive := receiveAll
	tradeVolGreaterThan := decimal.NewFromInt(0)
	mylog.Infof("priceStr: %s,amount: %s,tradeVol: %s", priceStr, amount.String(), receiveAll.String())
	// 本次交易额大于等于设定值
	if receiveAll.GreaterThan(tradeVolGreaterThan) {
		//判断累计到手金额是否已回本
		if realizedProfit.GreaterThanOrEqual(totalVolumeBuy) {
			// 如果累计到手金额已回本，则本次全部视为盈利
			vAmount := amount.Mul(decimal.NewFromFloat(0.6))
			vVol := receiveAll.Mul(decimal.NewFromFloat(0.6))

			vaultTip = vAmount.Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(toTokenDecimals))).BigInt()
			memeVaultInfo["冲狗基金回本"] = "冲狗基金-已回本，且盈利高于" + tradeVolGreaterThan.String() + "U"
			memeVaultInfo["本次交易盈利SOL"] = amount.String()
			memeVaultInfo["本次交易盈利金额"] = receiveAll.String()
			memeVaultInfo["用户本次收到的SOL"] = amount.Sub(vAmount).String()
			memeVaultInfo["用户本次收到金额"] = receiveAll.Sub(vVol).String()
			memeVaultInfo["冲狗基金本次收到SOL"] = vAmount.String()
			memeVaultInfo["冲狗基金本次收到金额"] = vVol.String()
			userReceive = receiveAll.Sub(vVol)
		} else {
			// 计算盈利部分   交易额-(总买入成本-已实现盈利)
			//累计接收未覆盖总买入成本，计算差额
			profit := receiveAll.Sub(totalVolumeBuy.Sub(realizedProfit))
			memeVaultInfo["本次交易盈利SOL"] = profit.Div(price).String()
			memeVaultInfo["本次交易盈利金额"] = profit.String()
			// 判断盈利金额是否大于设定值
			profitGreaterThan := decimal.NewFromFloat(0.1)
			if profit.GreaterThan(profitGreaterThan) {

				vVol := profit.Mul(decimal.NewFromFloat(0.6))
				vAmount := vVol.Div(price)
				fee := vAmount.Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(toTokenDecimals)))
				feeAmount := fee.BigInt()
				vaultTip = feeAmount
				memeVaultInfo["冲狗基金回本"] = "冲狗基金-已回本，且盈利高于" + tradeVolGreaterThan.String() + "U"
				memeVaultInfo["本次交易盈利SOL"] = profit.Div(price).String()
				memeVaultInfo["本次交易盈利金额"] = profit.String()
				memeVaultInfo["用户本次收到的SOL"] = amount.Sub(vAmount).String()
				memeVaultInfo["用户本次收到金额"] = receiveAll.Sub(vVol).String()
				memeVaultInfo["冲狗基金本次收到SOL"] = vAmount.String()
				memeVaultInfo["冲狗基金本次收到金额"] = vVol.String()

				userReceive = receiveAll.Sub(vVol)

			} else {
				memeVaultInfo["冲狗基金回本"] = "冲狗基金-未回本/盈利低于" + profitGreaterThan.String() + "U"
			}
		}

	} else {
		memeVaultInfo["冲狗基金回本"] = "冲狗基金交易额低于" + tradeVolGreaterThan.String() + "U"
	}
	memeVaultInfo["用户加本次累计卖出仓位总获得金额"] = realizedProfit.Add(userReceive).String()
	return userReceive, vaultTip, memeVaultInfo
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
	if req.JitoTipLamports.Sign() > 0 {
		JitoTip := make(map[string]interface{})
		JitoTip["jitoTipLamports"] = req.JitoTipLamports
		paramMap["prioritizationFeeLamports"] = JitoTip
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
