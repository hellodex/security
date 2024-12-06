package cmd

import (
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/system"
	"github.com/hellodex/HelloSecurity/wallet/enc"
	"golang.org/x/crypto/sha3"
)

type CommandHandler func(conn net.Conn, args []string)

type CommandRouter struct {
	handlers map[string]CommandHandler
}

func NewCommandRouter() *CommandRouter {
	return &CommandRouter{handlers: make(map[string]CommandHandler)}
}

func (r *CommandRouter) register(command string, handler CommandHandler) {
	r.handlers[command] = handler
}

func (r *CommandRouter) ParseCommands(command string) {
	if strings.HasPrefix(command, "hi") {
		r.register("hi", showWelcome)
	} else if strings.HasPrefix(command, "initSec") {
		r.register("initSec", initSec)
	} else {
		r.register("unknown", unknown)
	}
}

func (r *CommandRouter) Route(conn net.Conn, command string) {
	args := strings.Fields(command)
	if len(args) == 0 {
		return // 空命令
	}

	cmd := args[0]
	handler, ok := r.handlers[cmd]
	if !ok {
		conn.Write([]byte("Unknown command\n"))
		return
	}

	handler(conn, args[1:])
}

func showWelcome(conn net.Conn, args []string) {
	sb := "Wlecome To Hello Security\n"

	conn.Write([]byte(sb))
}

func hashHex(o string) string {
	hash := sha3.New256()
	hash.Write([]byte(o))
	hashedBytes := hash.Sum(nil)
	return hex.EncodeToString(hashedBytes)
}

func initSec(conn net.Conn, args []string) {
	if len(args) != 1 {
		conn.Write([]byte("unexpected params\n"))
		return
	}
	keyone := args[0]
	// val := crypto.Keccak256([]byte(keyone))
	// valHex := hex.EncodeToString(val)
	valHex := hashHex(keyone)

	db := system.GetDb()
	var result model.SysDes
	db.Model(&model.SysDes{}).Where("desk = ? and flag = ?", "Sys_Init", 0).Take(&result)
	if (result.ID) == 0 {
		m := model.SysDes{
			Desk:       "Sys_Init",
			Desv:       valHex,
			UpdateTime: time.Now(),
			Flag:       0,
		}
		db.Save(&m)
	} else {
		if valHex != result.Desv {
			conn.Write([]byte("wrong key setting\n"))
			return
		}
	}
	err := enc.GetEP().SetAESKey(keyone)
	if err != nil {
		conn.Write([]byte(fmt.Sprintf("wrong key setting with %s\n", err.Error())))
		return
	}
	conn.Write([]byte("success\n"))
}

func unknown(conn net.Conn, args []string) {
	conn.Write([]byte("unknown command\n"))
}
