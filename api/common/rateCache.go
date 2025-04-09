package common

import (
	"github.com/beefsack/go-rate"
	"github.com/jellydator/ttlcache/v3"
	"time"
)

var rateCache *ttlcache.Cache[string, *rate.RateLimiter]

func init() {
	rateCache = ttlcache.New[string, *rate.RateLimiter](ttlcache.WithTTL[string, *rate.RateLimiter](3 * time.Minute))
	go rateCache.Start()
}
func RateLimiterMoreThan(key string, limit uint64, duration time.Duration) bool {
	item := rateCache.Get(key)
	if item == nil || item.Value() == nil {
		rateCache.Set(key, rate.New(3, duration), 3*time.Minute)
		return false
	}
	rateLimiter := item.Value()
	ok, _ := rateLimiter.Try()
	return !ok
}
