package common

import (
	"github.com/hellodex/HelloSecurity/log"
	"github.com/sirupsen/logrus"
)

func GetLog() *logrus.Logger {
	return log.GetLogger()
}
