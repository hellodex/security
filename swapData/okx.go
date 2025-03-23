package swapData

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/codes"
	"github.com/hellodex/HelloSecurity/config"
	"github.com/hellodex/HelloSecurity/log"
	"github.com/klauspost/compress/gzhttp"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"time"
)

var cfg = config.GetConfig()
var method = "GET"

func GetSwapDataWithOpts(retries int, s map[string]interface{}, params *common.LimitOrderParam) (map[string]interface{}, SwapDataResult, error) {
	if params.ShouldOkx {
		req, response, err := GetSwapData(retries, s, params)
		result := SwapDataResult{
			Plat: codes.Okx,
			Data: response,
		}
		return req, result, err

	}
	if params.ChainCode != "SOLANA" {
		req, response, err := GetSwapDataByOxApi(retries, s, params)
		result := SwapDataResult{
			Plat: codes.Bsc0x,
			Data: response,
		}
		return req, result, err
	}
	if params.ChainCode == "SOLANA" {
		req, response, err := GetSwapDataByJupApi(retries, s, params)
		result := SwapDataResult{
			Plat: codes.Jup,
			Data: response,
		}
		return req, result, err
	}
	return map[string]interface{}{"msg": "Unsupported requests"}, SwapDataResult{
		Plat: codes.Bsc0x,
		Data: nil,
	}, errors.New("GetSwapDataWithOpts failed")
}
func GetSwapData(retries int, s map[string]interface{}, params *common.LimitOrderParam) (map[string]interface{}, OkxResponse, error) {
	api, response, err := SwapDataByOkxApi(params)
	swapmap := make(map[string]interface{})
	if err != nil {
		swapmap["req"] = api
		swapmap["res"] = response
		s["swapData"+strconv.Itoa(retries)] = swapmap
	} else {
		if response.Code == "0" {
			swapmap["req"] = api
			swapmap["res"] = response
			s["swapData"+strconv.Itoa(retries)] = swapmap
			return s, response, nil
		} else {
			swapmap["req"] = api
			swapmap["res"] = response
			s["swapData"+strconv.Itoa(retries)] = swapmap
			return s, response, errors.New("GetSwapDataFailed" + response.Msg)
		}

	}

	return s, response, err
}

func SwapDataByOkxApi(params *common.LimitOrderParam) (common.LimitOrderParam, OkxResponse, error) {
	maxRetries := cfg.Okxswap.MaxRetry
	retryCount := 0
	var okxRes OkxResponse
	params.CurrTime = time.Now().Format("2006-01-02 15:04:05.000")

	for retryCount < maxRetries {
		retryCount++
		isoString := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
		log.Logger.Print("isoString:", isoString)
		log.Logger.Printf("cfg AccessKey:%+v:", cfg)

		var apiUrl = cfg.Okxswap.Host + params.ReqUri + "&slippage=" + params.Slippage
		request, err := http.NewRequest("GET", apiUrl, nil)
		beSin := isoString + method + request.URL.RequestURI()
		h := hmac.New(sha256.New, []byte(cfg.Okxswap.Secret))
		h.Write([]byte(beSin))
		sign := base64.StdEncoding.EncodeToString(h.Sum(nil))

		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("contentType", "application/json")
		request.Header.Set("OK-ACCESS-KEY", cfg.Okxswap.AccessKey)
		request.Header.Set("OK-ACCESS-PASSPHRASE", cfg.Okxswap.AccessPassphrase)
		request.Header.Set("OK-ACCESS-PROJECT", cfg.Okxswap.Project)
		request.Header.Set("OK-ACCESS-TIMESTAMP", isoString)
		request.Header.Set("OK-ACCESS-SIGN", sign)
		resp, err := HTTPClient.Do(request)

		if err != nil {
			if err != nil {
				log.Logger.Error("SwapDataHTTPClient.Do(request) err" + err.Error())
				continue
			}
			time.Sleep(50 * time.Millisecond)
			continue
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Logger.Error("SwapDataByOkxApi err" + err.Error())
			time.Sleep(50 * time.Millisecond)
			continue
		}
		bodyReadErr := resp.Body.Close()
		if bodyReadErr == nil {
			jsonErr := json.Unmarshal(body, &okxRes)
			if jsonErr != nil {
				log.Error("SwapDataByOkxApi json err" + jsonErr.Error())
			}
			if jsonErr == nil {
				if okxRes.Code == "82116" {
					time.Sleep(50 * time.Millisecond)
					continue
				}
				okxRes.RetryCount = retryCount
				okxRes.ReqUri = apiUrl

				return *params, okxRes, nil
			}
		}

	}
	okxRes.Code = "400"
	okxRes.Msg = "GetSwapDataFailed"
	okxRes.RetryCount = retryCount
	return *params, okxRes, nil
}

type OkxResponse struct {
	RetryCount int    `json:"retryCount"`
	Code       string `json:"code"`
	Data       []struct {
		RouterResult struct {
			ChainID       string `json:"chainId"`
			DexRouterList []struct {
				Router        string `json:"router"`
				RouterPercent string `json:"routerPercent"`
				SubRouterList []struct {
					DexProtocol []struct {
						DexName string `json:"dexName"`
						Percent string `json:"percent"`
					} `json:"dexProtocol"`
					FromToken struct {
						Decimal              string `json:"decimal"`
						IsHoneyPot           bool   `json:"isHoneyPot"`
						TaxRate              string `json:"taxRate"`
						TokenContractAddress string `json:"tokenContractAddress"`
						TokenSymbol          string `json:"tokenSymbol"`
						TokenUnitPrice       string `json:"tokenUnitPrice"`
					} `json:"fromToken"`
					ToToken struct {
						Decimal              string `json:"decimal"`
						IsHoneyPot           bool   `json:"isHoneyPot"`
						TaxRate              string `json:"taxRate"`
						TokenContractAddress string `json:"tokenContractAddress"`
						TokenSymbol          string `json:"tokenSymbol"`
						TokenUnitPrice       string `json:"tokenUnitPrice"`
					} `json:"toToken"`
				} `json:"subRouterList"`
			} `json:"dexRouterList"`
			EstimateGasFee string `json:"estimateGasFee"`
			FromToken      struct {
				Decimal              string `json:"decimal"`
				IsHoneyPot           bool   `json:"isHoneyPot"`
				TaxRate              string `json:"taxRate"`
				TokenContractAddress string `json:"tokenContractAddress"`
				TokenSymbol          string `json:"tokenSymbol"`
				TokenUnitPrice       string `json:"tokenUnitPrice"`
			} `json:"fromToken"`
			FromTokenAmount       string `json:"fromTokenAmount"`
			PriceImpactPercentage string `json:"priceImpactPercentage"`
			QuoteCompareList      []struct {
				AmountOut string `json:"amountOut"`
				DexLogo   string `json:"dexLogo"`
				DexName   string `json:"dexName"`
				TradeFee  string `json:"tradeFee"`
			} `json:"quoteCompareList"`
			ToToken struct {
				Decimal              string `json:"decimal"`
				IsHoneyPot           bool   `json:"isHoneyPot"`
				TaxRate              string `json:"taxRate"`
				TokenContractAddress string `json:"tokenContractAddress"`
				TokenSymbol          string `json:"tokenSymbol"`
				TokenUnitPrice       string `json:"tokenUnitPrice"`
			} `json:"toToken"`
			ToTokenAmount string `json:"toTokenAmount"`
			TradeFee      string `json:"tradeFee"`
		} `json:"routerResult"`
		Tx struct {
			Data                 string   `json:"data"`
			From                 string   `json:"from"`
			Gas                  string   `json:"gas"`
			GasPrice             string   `json:"gasPrice"`
			MaxPriorityFeePerGas string   `json:"maxPriorityFeePerGas"`
			MinReceiveAmount     string   `json:"minReceiveAmount"`
			SignatureData        []string `json:"signatureData"`
			To                   string   `json:"to"`
			Value                string   `json:"value"`
		} `json:"tx"`
	} `json:"data"`
	Msg    string `json:"msg"`
	ReqUri string `json:"reqUri"`
}

func newHTTPTransport() *http.Transport {

	return &http.Transport{
		IdleConnTimeout:     defaultTimeout,
		MaxConnsPerHost:     defaultMaxIdleConnsPerHost,
		MaxIdleConnsPerHost: defaultMaxIdleConnsPerHost,
		MaxIdleConns:        defaultMaxIdleConns,
		Proxy: func(req *http.Request) (*url.URL, error) {
			// 对某些域名或 URL 使用代理
			if req.URL.Host == "www.okx.com" && runtime.GOOS == "windows" {
				return url.Parse("http://127.0.0.1:7890") // 指定代理
			}
			// 不使用代理
			return nil, nil
		},
		DialContext: (&net.Dialer{
			Timeout:   300 * time.Second,
			KeepAlive: defaultKeepAlive,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   20 * time.Second,
		ExpectContinueTimeout: 10 * time.Second,
	}
}

type SwapDataResult struct {
	Plat string      `json:"plat"`
	Data interface{} `json:"swapData"`
}

var (
	defaultMaxIdleConns        = 3000
	defaultMaxIdleConnsPerHost = 3000
	defaultTimeout             = 5 * time.Minute
	defaultKeepAlive           = 5 * time.Minute
)
var HTTPClient = &http.Client{
	Timeout:   defaultTimeout,
	Transport: gzhttp.Transport(newHTTPTransport()),
	//Transport: &CurlTransport{
	//	Transport: http.DefaultTransport,
	//},
}
