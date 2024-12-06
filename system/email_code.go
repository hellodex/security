package system

import (
	myLog1 "github.com/hellodex/HelloSecurity/log"
	"github.com/jellydator/ttlcache/v3"
	"golang.org/x/exp/rand"
	"strconv"
	"time"
)

var CodeCache *ttlcache.Cache[string, string]

func GenCode(addr string, captchaType string) string {
	rand.Seed(uint64(time.Now().UnixNano()))
	var randomString string
	for i := 0; i < 6; i++ {
		digit := rand.Intn(10)
		randomString += strconv.Itoa(digit)
	}
	myLog1.Infof("email code for %s: %s,key: %s", addr, randomString, addr+captchaType)
	CodeCache.Set(addr+captchaType, randomString, 3*time.Minute)
	return randomString
}

func VerifyCode(addr, code string) bool {
	myLog1.Infof("email VerifyCode for %s: %s ", addr, code)
	item := CodeCache.Get(addr)
	if item == nil {
		return false
	}
	cachedCode := item.Value()
	suceess := cachedCode == code
	if suceess {
		CodeCache.Delete(addr)
	}
	return suceess
}

func init() {
	CodeCache = ttlcache.New[string, string](ttlcache.WithTTL[string, string](3 * time.Minute))
	go CodeCache.Start()
}
