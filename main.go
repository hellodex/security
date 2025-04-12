package main

import (
	"bufio"
	"flag"
	"fmt"
	"runtime/pprof"

	"net"
	"net/http"

	"os"

	"strings"
	"sync"

	"github.com/gorilla/websocket"
	router "github.com/hellodex/HelloSecurity/api"
	"github.com/hellodex/HelloSecurity/cmd"
	"github.com/hellodex/HelloSecurity/config"
	log "github.com/hellodex/HelloSecurity/log"
	"github.com/hellodex/HelloSecurity/runtime"
)

var logger = log.Logger
var wg sync.WaitGroup

func initWebSocket(url string) *websocket.Conn {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		logger.Fatalf("Failed to connect to WebSocket at %s: %v", url, err)
	}
	return conn
}

func handleMgrConnection(conn net.Conn) {
	logger.Println("New client connected:", conn.RemoteAddr().String())
	defer conn.Close()

	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			//logger.Println("Client disconnected:", conn.RemoteAddr().String())
			break
		}

		message = message[:len(message)-1] // 去除换行符
		message = strings.TrimRight(message, "\r\n")
		//logger.Printf("Received command from %s: %s\n", conn.RemoteAddr().String(), message)
		if len(message) > 0 {
			if message == "exit" {
				logger.Println("Exiting Connection.")
				_ = conn.Close()
			} else {
				router := cmd.NewCommandRouter()
				router.ParseCommands(message)
				router.Route(conn, message)
			}
		}
	}
}

func main() {
	conf := config.GetConfig()

	defer log.LogFile.Close()

	defer func() {
		if r := recover(); r != nil {
			log.Logger.WithField("panic", r).Error("Application panic")
		}
		if log.LogFile != nil {
			log.LogFile.Close()
		}
	}()

	portInt := flag.Int("p", conf.Cmd.Port, "The Console Port")
	modeStr := flag.String("mode", "slot", "Mode of system (token or slot)")
	flag.Parse()

	runtime.InitSys(*modeStr)

	listenAddr := fmt.Sprintf("%s:%d", conf.Cmd.Host, *portInt)
	server, err := net.Listen("tcp", listenAddr)
	if err != nil {
		logger.Fatal("system error:", err)
	}

	defer server.Close()

	logger.Printf("Bot starting, and the console port is %d", *portInt)

	// recordHeap()
	// recordHttp()

	go func() {
		for {
			conn, err := server.Accept()
			if err != nil {
				fmt.Println("Error accepting connection:", err)
				continue
			}
			wg.Add(1)
			go handleMgrConnection(conn)
		}
	}()

	router.Init()
}

func recordHeap() {
	f, err := os.Create("/app/hellosol/solHeap.prof")
	if err != nil {
		logger.Errorln("create heap record error", err)
		return
	}
	defer f.Close()

	// 开始内存Profile
	err = pprof.WriteHeapProfile(f)
	if err != nil {
		logger.Errorln("record heap error", err)
	}
}

func recordHttp() {
	go func() {
		logger.Println(http.ListenAndServe("0.0.0.0:6060", nil))
	}()
}
