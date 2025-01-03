package http

import (
	"github.com/gin-gonic/gin"
	"github.com/hellodex/HelloSecurity/api/http/controller"
	"github.com/hellodex/HelloSecurity/api/interceptor"
)

func Routers(e *gin.RouterGroup) {

	sysGroup := e.Group("/auth", interceptor.HttpInterceptor())
	//sysGroup := e.Group("/auth")
	sysGroup.POST("/wallet/create/byChain", controller.CreateWallet)
	sysGroup.POST("/wallet/create/batch", controller.CreateBatchWallet)
	//sysGroup.POST("/wallet/sig", controller.Sig)
	sysGroup.POST("/wallet/sig", controller.AuthSig)
	sysGroup.POST("/wallet/list", controller.List)
	//sysGroup.POST("/wallet/transfer", controller.Transfer)
	sysGroup.POST("/wallet/transfer", controller.AuthTransfer)

	sysGroup.POST("/sys/sendMessage", controller.SendEmail)
	sysGroup.POST("/sys/registerCheck", controller.VerifyCode)

	sysGroup.POST("/sys/initSeg", controller.InitKeySeg)
	sysGroup.POST("/sys/testRun", controller.TestRun)

	sysGroup.POST("/wallet/createLimitKey", controller.CreateLimitKey) //创建限价单密钥
	sysGroup.POST("/wallet/delLimitKey", controller.DelLimitOrderKeys) //删除限价单密钥
	sysGroup.POST("/wallet/loginOut", controller.DelWalletKeys)        //创建市价单密钥

	sysGroup.POST("/wallet/authCreateBatch", controller.AuthCreateBatchWallet) //批量创建钱包
	//sysGroup.POST("/wallet/authCreateBatchTg", controller.AuthCreateBatchTgWallet)   //批量创建钱包
	sysGroup.POST("/wallet/authCreateBatchTg1", controller.AuthCreateBatchTgWallet1) //批量创建钱包
}
