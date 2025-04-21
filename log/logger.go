package log

import (
	"github.com/sirupsen/logrus"
	"io"
	"os"
)

var Logger *logrus.Logger
var LogFile *os.File

type FileHook struct {
	Writer    io.Writer
	LogLevels []logrus.Level
}

func (hook *FileHook) Fire(entry *logrus.Entry) error {
	// 手动捕获调用者信息
	//if entry.HasCaller() {
	//	// 根据调用堆栈深度获取实际调用者信息
	//	pc, file, line, ok := runtime.Caller(8) // 调整堆栈深度
	//	if ok {
	//		funcName := runtime.FuncForPC(pc).Name()
	//		if !strings.Contains(funcName, "HelloSecurity/log.") {
	//			entry.Records["file"] = fmt.Sprintf("%s:%d", file, line)
	//			entry.Records["func"] = funcName
	//		}
	//
	//	} else {
	//		entry.Records["file"] = "unknown"
	//		entry.Records["func"] = "unknown"
	//	}
	//}

	// 使用自定义格式化器格式化日志输出
	formatter := &logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	}

	// 将格式化后的日志写入文件
	line, err := formatter.Format(entry)
	if err != nil {
		return err
	}

	_, err = hook.Writer.Write(line)
	if err != nil {
		//fmt.Printf("Error writing log: %v\n", err)
	} else {
		//fmt.Printf("Successfully wrote log entry: %s\n", string(line))
	}
	return err
}

func (hook *FileHook) Levels() []logrus.Level {
	return hook.LogLevels
}

func InitLogger(path string) {
	Logger = logrus.New()
	Logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	Logger.SetLevel(logrus.InfoLevel)
	Logger.SetReportCaller(true) // Enable reporting caller info

	infoFile, err := os.OpenFile(path+"\\info.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		Logger.Fatal(err)
	}

	errorFile, err := os.OpenFile(path+"\\error.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)

	if err != nil {
		Logger.Fatal(err)
	}

	Logger.SetOutput(os.Stdout)

	Logger.AddHook(&FileHook{
		Writer: infoFile,
		LogLevels: []logrus.Level{
			logrus.InfoLevel,
			logrus.WarnLevel,
		},
	})

	Logger.AddHook(&FileHook{
		Writer: errorFile,
		LogLevels: []logrus.Level{
			logrus.ErrorLevel,
			logrus.FatalLevel,
			logrus.PanicLevel,
		},
	})
	Logger.Info("Logger initialized")
}

func Info(args ...interface{}) {
	Logger.Info(args...)
}

func Infof(format string, args ...interface{}) {
	Logger.Infof(format, args...)
}

func Error(args ...interface{}) {
	Logger.Error(args...)
}

func Errorf(format string, args ...interface{}) {
	Logger.Errorf(format, args...)
}

func Fatal(args ...interface{}) {
	Logger.Fatal(args...)
}
func GetLogger() *logrus.Logger {
	return Logger
}
