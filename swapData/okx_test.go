package swapData

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/log"
	"math/big"
	"testing"
)

func Test_okx_test(t *testing.T) {
	call := make(map[string]interface{})
	request := common.LimitOrderParam{
		Amount:            new(big.Int).SetUint64(1267170),
		FromTokenAddress:  "11111111111111111111111111111111",
		Slippage:          "0.05",
		ChainCode:         "SOLANA",
		OrderNo:           "123456789",
		UserWalletAddress: "23PGuJNLTvFWSAYS1S6HGL3gg7fgPsF5GbiXaC4puCSZ",
		ReqUri:            "/api/v5/dex/aggregator/swap?chainId=501&amount=1267170&fromTokenAddress=11111111111111111111111111111111&toTokenAddress=GxdTh6udNstGmLLk9ztBb6bkrms7oLbrJp5yzUaVpump&priceImpactProtectionPercentage=1&userWalletAddress=23PGuJNLTvFWSAYS1S6HGL3gg7fgPsF5GbiXaC4puCSZ&feePercent=1&fromTokenReferrerWalletAddress=39sXPZ4rD86nA3YoS6YgF5sdutHotL87U6eQnADFRkRE",
	}
	data, response, _ := GetSwapData(2, call, &request)
	spew.Dump(data)
	log.Logger.Info("response: ")
	spew.Dump(response)

}
func Test_Jup_GetSwapData(t *testing.T) {
	call := make(map[string]interface{})
	request := common.LimitOrderParam{
		Amount:                  new(big.Int).SetUint64(2111568),
		FromTokenAddress:        "So11111111111111111111111111111111111111112",
		ToTokenAddress:          "GxdTh6udNstGmLLk9ztBb6bkrms7oLbrJp5yzUaVpump",
		Slippage:                "0.05",
		ChainCode:               "SOLANA",
		OrderNo:                 "123456789",
		UserWalletAddress:       "KERxu1WdAfziZbmRkZnpj7mUgyJrLGdYC7d1VMwPR25",
		ShouldOkx:               false,
		DynamicSlippage:         false,
		DynamicComputeUnitLimit: false,
	}
	data, response, _ := GetSwapDataWithOpts(2, call, &request)
	spew.Dump(data)
	log.Logger.Info("response: ")
	spew.Dump(response)

}
