package swapData

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/codes"
	"github.com/hellodex/HelloSecurity/config"
	"github.com/hellodex/HelloSecurity/log"
	"github.com/klauspost/compress/gzhttp"
	"github.com/shopspring/decimal"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"time"
)

var cfg = config.GetConfig()
var method = "GET"
var mylog = log.GetLogger()

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
			if len(response.Data) > 0 {
				userReceive, vaultTip, memeVaultInfo := memeVaultTip(response.Data[0].Tx.MinReceiveAmount,
					response.Data[0].RouterResult.ToToken.TokenContractAddress,
					params.Amount, params.FromTokenDecimals, params.ToTokenDecimals,
					params.UserWalletAddress, params.FromTokenAddress, params.TotalVolumeBuy, params.RealizedProfit, params.AvgPrice)
				response.UserReceive = userReceive
				response.VaultTip = vaultTip
				response.MemeVaultInfo = memeVaultInfo
			}
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
func SendSolTxByOkxApi(ctx context.Context, tx *solana.Transaction) (solana.Signature, error) {
	txBase64, err := tx.ToBase64()
	mylog.Info("okx 上链transaction content: ", txBase64, err)
	maxRetries := cfg.Okxswap.MaxRetry
	retryCount := 0
	var okxRes OkxTxResponse

	for retryCount < maxRetries {
		retryCount++
		isoString := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

		req := make(map[string]interface{})
		req["chainIndex"] = "501"
		req["address"] = tx.Message.AccountKeys[0].String()
		req["signedTx"] = txBase64
		req["extraData"] = map[string]interface{}{
			"enableMevProtection": true,
			"jitoSignedTx":        txBase64,
		}
		jsonData, err := json.Marshal(req)
		if err != nil {
			mylog.Info("okx 组装参数报错")
		}

		//var apiUrl = cfg.Okxswap.Host + "/api/v5/dex/pre-transaction/broadcast-transaction"
		var apiUrl = "https://web3.okx.com/api/v5/dex/pre-transaction/broadcast-transaction"
		request, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonData))
		beSin := isoString + "POST" + request.URL.RequestURI()
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
				log.Logger.Errorf("OKX sendTx .Do(request) err:%v", err.Error())
				continue
			}
			time.Sleep(50 * time.Millisecond)
			continue
		}
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(fmt.Sprintf("OKX sendTx req:%v resp:%s", req, string(body)))

		bodyReadErr := resp.Body.Close()
		if bodyReadErr == nil {
			if err := json.Unmarshal(body, &okxRes); err == nil && okxRes.Code == "0" {

				sig, err := solana.SignatureFromBase58(okxRes.Data[0].TxHash)
				if err != nil {
					return solana.Signature{}, fmt.Errorf("OKX sendTx  invalid signature format: %v", err)
				}
				return sig, nil
			}

		}

	}

	return solana.Signature{}, errors.New("SendSolTxByOkxApi failed")
}

type OkxTxResponse struct {
	Code string `json:"code"`
	Data []struct {
		OrderId string `json:"orderId"`
		TxHash  string `json:"txHash"`
	} `json:"data"`
	Msg string       `json:"msg,omitempty"`
	Err *interface{} `json:"err,omitempty"`
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
	Msg           string                 `json:"msg"`
	ReqUri        string                 `json:"reqUri"`
	UserReceive   decimal.Decimal        `json:"userReceive"`
	VaultTip      *big.Int               `json:"vaultTip"`
	MemeVaultInfo map[string]interface{} `json:"memeVaultInfo"`
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
