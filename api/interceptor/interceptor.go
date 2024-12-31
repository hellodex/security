package interceptor

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/codes"
	"github.com/hellodex/HelloSecurity/log"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/system"
)

var exception = []string{""}

var cacheKeys []model.SysChannel
var cacheValid = time.Now().Unix()

const timeRange = 60

var wg sync.Mutex

// http 请求拦截器
func HttpInterceptor() gin.HandlerFunc {
	return func(c *gin.Context) {
		queryKeys()
		app_id := c.Request.Header.Get("APP_ID")

		auth_token := c.Request.Header.Get("SIG")
		ts := c.Request.Header.Get("TS")
		ver := c.Request.Header.Get("VER")
		request_id := c.Request.Header.Get("REQUEST_ID")
		c.Set("appId", app_id)
		c.Set("sig", auth_token)
		var targetChannel *model.SysChannel
		for _, v := range cacheKeys {
			if v.AppID == app_id {
				targetChannel = &v
			}
		}
		if targetChannel == nil {
			c.Abort()
			c.JSON(http.StatusOK, common.Response{
				Code:      codes.CODE_ERR_APPID_INVALID,
				Msg:       "app_id invalid",
				Timestamp: time.Now().Unix(),
			})
			return
		}
		if ok, code := targetChannel.Verify(app_id+request_id+ts+ver, auth_token); !ok {
			c.Abort()
			c.JSON(http.StatusOK, common.Response{
				Code:      int64(code),
				Msg:       "sig or key params wrong or empty",
				Timestamp: time.Now().Unix(),
			})
			return
		}
		c.Set("APP_ID", app_id)
		c.Set("REQUEST_ID", request_id)
		c.Set("TS", ts)
		c.Next()
	}
}

func queryKeys() []model.SysChannel {
	if len(cacheKeys) > 0 && time.Now().Unix()-cacheValid <= (timeRange) {
		return cacheKeys
	}
	db := system.GetDb()
	var result []model.SysChannel
	err := db.Model(&model.SysChannel{}).Where("status = ?", "00").Find(&result).Error
	if err != nil {
		log.Error("Channel Query Error:", err)
		return cacheKeys
	}

	wg.Lock()
	cacheKeys = result
	cacheValid = time.Now().Unix()
	wg.Unlock()
	return cacheKeys
}
