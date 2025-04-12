package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/go-co-op/gocron"
	"math"
	"testing"
	"time"
)

func TestHash(t *testing.T) {
	password := "123456"
	hashPassByte := sha256.Sum256([]byte(password))
	hashPass := hex.EncodeToString(hashPassByte[:])

	fmt.Println(hashPass)
}

func TestFloat(t *testing.T) {
	v := 86
	vs := float64(v) / 100
	s := math.Round(vs*10) / 10
	fmt.Println(s)
}
func TestFloat2(t *testing.T) {
	scheduler := gocron.NewScheduler(time.Local)
	retries := 0
	scheduler.Every(500).Tag("waitForTx").Millisecond().SingletonMode().LimitRunsTo(10).Do(func() {
		retries++
		time.Sleep(time.Second)
		fmt.Printf("waitForTx  retries: %d\n", retries)

		if retries > 5 {
			//scheduler.StopBlockingChan()
			fmt.Printf("scheduler stopped after %d retries\n", retries)
		}

	})
	scheduler.StartBlocking()
	fmt.Printf("scheduler started with %d retries\n", retries)
}
