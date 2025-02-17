package common

import (
	"github.com/jellydator/ttlcache/v3"
	"sync"
	"time"
)

var lockCache *ttlcache.Cache[string, *MyLock]

func init() {
	lockCache = ttlcache.New[string, *MyLock](ttlcache.WithTTL[string, *MyLock](3 * time.Minute))
	go lockCache.Start()
}
func GetLock(key string) *MyLock {
	item := lockCache.Get(key)
	if item == nil || item.Value() == nil {
		mutex := MyLock{Lock: sync.Mutex{}}
		set := lockCache.Set(key, &mutex, 3*time.Minute)
		return set.Value()
	}
	return item.Value()
}
func GetLockWithTTL(key string, ttl time.Duration) *MyLock {
	if lockCache == nil {

	}
	item := lockCache.Get(key)
	if item == nil || item.Value() == nil {
		mutex := MyLock{Lock: sync.Mutex{}}
		set := lockCache.Set(key, &mutex, ttl)
		return set.Value()
	}
	return item.Value()
}

type MyLock struct {
	Lock sync.Mutex
}
