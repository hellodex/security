package controller

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/codes"
	"github.com/hellodex/HelloSecurity/wallet/enc"
)

const methodName = "Decrypt"

type KeyInitReq struct {
	Val string `json:"val"`
}

type TestRunReq struct {
	Val    string `json:"val"`
	TryDec string `json:"try_dec"`
}

func InitKeySeg(c *gin.Context) {
	var req KeyInitReq
	res := common.Response{}
	res.Timestamp = time.Now().Unix()

	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "Invalid Request"
		c.JSON(http.StatusOK, res)
		return
	}

	if len(req.Val) == 0 {
		res.Code = codes.CODES_ERR_PARA_EMPTY
		res.Msg = "Invalid Parameter"
		c.JSON(http.StatusOK, res)
		return
	}

	coverOk := enc.Porter().Recovered()
	if !coverOk {
		ok, err := enc.Porter().SetSegKey(req.Val)
		if err != nil {
			res.Code = codes.CODE_ERR_UNKNOWN
			res.Msg = fmt.Sprintf("init seg: %s", err.Error())
			c.JSON(http.StatusOK, res)
			return
		}
		if ok {
			coverOk = ok
		}
	}

	res.Code = codes.CODE_SUCCESS
	res.Msg = "success"
	res.Data = coverOk

	c.JSON(http.StatusOK, res)
}

func TestRun(c *gin.Context) {
	var req TestRunReq
	res := common.Response{}
	res.Timestamp = time.Now().Unix()

	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "Invalid Request"
		c.JSON(http.StatusOK, res)
		return
	}

	if len(req.Val) == 0 {
		res.Code = codes.CODES_ERR_PARA_EMPTY
		res.Msg = "Invalid Parameter"
		c.JSON(http.StatusOK, res)
		return
	}

	coverOk := enc.Porter().Recovered()
	if !coverOk {
		res.Code = codes.CODE_ERR_UNKNOWN
		res.Msg = "waiting for more seg for recovering"
		c.JSON(http.StatusOK, res)
		return
	}

	enb, err := enc.Porter().Encrypt([]byte(req.Val))

	if err != nil {
		res.Code = codes.CODE_ERR_UNKNOWN
		res.Msg = fmt.Sprintf("encrypt error: %s", err.Error())
		c.JSON(http.StatusOK, res)
		return
	}

	encryptVal := base64.StdEncoding.EncodeToString(enb)
	var decryptVal, tryDec []byte

	encPort := enc.Porter()
	hasMethod := hasDecryptMethod(encPort, methodName)
	if hasMethod {
		envb, _ := base64.StdEncoding.DecodeString(encryptVal)
		decryptVal, _ = callDecrypt(encPort, envb[12:], envb[:12])

		if len(req.TryDec) > 0 {
			tryDec, err = base64.StdEncoding.DecodeString(req.TryDec)
			if err != nil {
				tryDec = []byte("error: base64 decode error")
			} else if len(tryDec) <= 12 {
				tryDec = []byte("error: try enc length error")
			} else {
				tryDec, err = callDecrypt(encPort, tryDec[12:], tryDec[:12])
				if err != nil {
					tryDec = []byte("error: try decrypt error: " + err.Error())
				}
			}
		}
	} else {
		decryptVal = []byte("unsupport decrypt for security")
		tryDec = []byte("unsupport decrypt for security")
	}

	res.Code = codes.CODE_SUCCESS
	res.Msg = ""
	res.Data = struct {
		Encrypt    string
		Decrypt    string
		TryDecrypt string
	}{
		Encrypt:    encryptVal,
		Decrypt:    string(decryptVal),
		TryDecrypt: string(tryDec),
	}
	c.JSON(http.StatusOK, res)
}

func hasDecryptMethod(obj interface{}, methodName string) bool {
	objType := reflect.TypeOf(obj)

	method, ok := objType.MethodByName(methodName)
	if !ok {
		return false
	}

	if method.Type.NumIn() != 3 || method.Type.NumOut() != 2 {
		return false
	}

	if method.Type.In(1) != reflect.TypeOf([]byte{}) || method.Type.In(2) != reflect.TypeOf([]byte{}) {
		return false
	}

	if method.Type.Out(0) != reflect.TypeOf([]byte{}) || method.Type.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
		return false
	}

	return true
}

func callDecrypt(obj interface{}, ciphertext, nonce []byte) ([]byte, error) {
	method := reflect.ValueOf(obj).MethodByName("Decrypt")

	results := method.Call([]reflect.Value{reflect.ValueOf(ciphertext), reflect.ValueOf(nonce)})

	decryptedBytes := results[0].Interface().([]byte)
	var err error
	if !results[1].IsNil() {
		err = results[1].Interface().(error)
	}

	return decryptedBytes, err
}
