package swapData

import (
	"encoding/json"
	"fmt"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/shopspring/decimal"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

func GetSwapDataByOxApi(retries int, s map[string]interface{}, params *common.LimitOrderParam) (map[string]interface{}, map[string]interface{}, error) {
	api, response, err := getSwapDate0xAPI(params)
	if response != nil {
		t, ex := response["transaction"]
		if ex {
			tr := t.(map[string]interface{})
			data, ex := tr["data"]
			if ex {
				response["singData"] = data.(string)
			}
			to, ex := tr["to"]
			if ex {
				response["to"] = to.(string)
			}
			value, ex := tr["value"]
			if ex {
				response["value"] = value.(string)
			}
			gas, ex := tr["gas"]
			if ex {
				response["gas"] = gas.(string)
			}
			gasPrice, ex := tr["gasPrice"]
			if ex {
				response["gasPrice"] = gasPrice.(string)
			}
		}
	}
	swapmap := make(map[string]interface{})
	swapmap["req"] = api
	swapmap["res"] = response
	s["swapData"+strconv.Itoa(retries)] = swapmap
	return s, response, err
}

// getSwapDate retrieves swap data from the 0x API based on the provided parameters.
func getSwapDate0xAPI(reqParams *common.LimitOrderParam) (common.LimitOrderParam, map[string]interface{}, error) {
	// Build query parameters
	slippage, err := decimal.NewFromString(reqParams.Slippage)
	if err != nil {
		slippage = decimal.NewFromFloat(0.05) // Default slippage to 1%
	}

	params := url.Values{}
	params.Add("sellAmount", reqParams.Amount.String())
	params.Add("chainId", reqParams.ChainId)
	params.Add("sellToken", reqParams.FromTokenAddress)
	params.Add("buyToken", reqParams.ToTokenAddress)
	params.Add("slippageBps", slippage.Mul(decimal.NewFromInt(10000)).String()) // Slippage, e.g., 100 means 1%
	params.Add("taker", reqParams.UserWalletAddress)
	params.Add("swapFeeBps", "100")
	params.Add("swapFeeRecipient", "0xf720Cd15EAd762290539CF4b23622E31B1be27e7") // Replace with actual constant in production
	params.Add("swapFeeToken", reqParams.FeeToken)

	// Define request headers
	headers := map[string]string{
		"0x-api-key": cfg.BscOxKey,
		"0x-version": "v2",
	}

	// Create HTTP client and request
	client := &http.Client{}
	req, err := http.NewRequest("GET", cfg.BscOxKeyHost+"/swap/allowance-holder/quote?"+params.Encode(), nil)
	if err != nil {
		return *reqParams, map[string]interface{}{}, fmt.Errorf("failed to create request: %v", err)
	}

	// Add headers to the request
	for key, value := range headers {
		req.Header.Add(key, value)
	}

	// Attempt the request up to 3 times
	var result string
	for i := 0; i < 3; i++ {
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Attempt %d failed: %v", i+1, err)
			continue
		}
		defer resp.Body.Close()

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

	// Check if a valid response was received
	if result == "" || result == "{}" {
		return *reqParams, map[string]interface{}{}, fmt.Errorf("failed to get valid response after 3 attempts")
	}

	// Parse the JSON response
	var data map[string]interface{}
	err = json.Unmarshal([]byte(result), &data)
	if err != nil {
		return *reqParams, map[string]interface{}{}, fmt.Errorf("failed to parse JSON response: %v", err)
	}

	return *reqParams, data, nil
}
