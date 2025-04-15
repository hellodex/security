package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/go-co-op/gocron"
	"github.com/hellodex/HelloSecurity/api/common"
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
	scheduler.Every(500).Millisecond().SingletonMode().LimitRunsTo(10).Tag("waitForTx").Do(func() {
		retries++
		time.Sleep(time.Second)
		fmt.Printf("waitForTx  retries: %d\n", retries)

		if retries > 5 {

			scheduler.RemoveByTag("waitForTx")
			scheduler.Clear()
			scheduler.StopBlockingChan()

			fmt.Printf("scheduler !!!!! after %d retries\n", retries)
		}

	})
	scheduler.StartBlocking()
	scheduler.Clear()
	fmt.Printf("scheduler started with %d retries\n", retries)
	fmt.Printf("    with %d retries\n", retries)
	for {
		time.Sleep(time.Second)
	}
}

func TestTwoFA(t *testing.T) {
	secret := common.TwoFACreateSecret(16, "aaa")
	fmt.Printf("secret: %+v", secret)
}
func TestTwoFA1(t *testing.T) {
	secret := common.TwoFAVerifyCode("VGYDKVS3SBLSF3OS", "131354", 0)
	fmt.Print("secret: --------->", secret)
	fmt.Print()
}
