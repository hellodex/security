package http

import (
	"github.com/gin-gonic/gin"
	"github.com/hellodex/HelloSecurity/api/http/controller"
	"github.com/hellodex/HelloSecurity/api/interceptor"
)

func Routers(e *gin.RouterGroup) {

	sysGroup := e.Group("/auth", interceptor.HttpInterceptor())

	sysGroup.POST("/wallet/sig", controller.AuthSig)
	sysGroup.POST("/wallet/transfer", controller.AuthTransfer)
	sysGroup.POST("/wallet/adminTransfer", controller.AuthAdminTransfer)

	sysGroup.POST("/sys/sendMessage", controller.SendEmail)
	sysGroup.POST("/sys/sendMessageV2", controller.SendEmailV2)

	sysGroup.POST("/sys/initSeg", controller.InitKeySeg)
	sysGroup.POST("/sys/testRun", controller.TestRun)

	sysGroup.POST("/wallet/createLimitKey", controller.CreateLimitKey) //创建限价单密钥
	sysGroup.POST("/wallet/delLimitKey", controller.DelLimitOrderKeys) //删除限价单密钥

	sysGroup.POST("/wallet/getUserLoginToken", controller.GetUserLoginToken)       //获取tg用户登录token
	sysGroup.POST("/wallet/verifyUserLoginToken", controller.VerifyUserLoginToken) //验证tg用户登录token
	sysGroup.POST("/wallet/AuthCloseAllAta", controller.AuthCloseAllAta)           //批量关闭所有余额为0的ata
	sysGroup.POST("/wallet/forceCloseAtas", controller.AuthForceCloseAll)          //批量烧币关闭所有余额不是0的ata

	sysGroup.POST("/user/login", controller.AuthUserLogin)                     //
	sysGroup.POST("/user/loginCheek", controller.AuthUserLoginCheek)           //
	sysGroup.POST("/user/register", controller.AuthUserRegister)               //
	sysGroup.POST("/user/AuthUserLoginCancel", controller.AuthUserLoginCancel) //
	sysGroup.POST("/user/AuthUserModifyPwd", controller.AuthUserModifyPwd)     //
	sysGroup.POST("/user/2faVerify", controller.AuthAdmin2FAVerify)            //

	// vaultSuppor 历史数据查询
	sysGroup.POST("/meme/vaultSupportList", controller.VaultSupportList) //
	// vaultSuppor 单个用户数据查询
	sysGroup.POST("/meme/vaultSupportListByUUID", controller.MemeVaultSupportListByUUID) //
	// vault 更新
	sysGroup.POST("/meme/vaultUpdate", controller.MemeVaultUpdate) //
	// vault 新增
	sysGroup.POST("/meme/vaultAdd", controller.MemeVaultAdd) //
	// vault 列表查询
	sysGroup.POST("/meme/vaultList", controller.MemeVaultList) //
	//领取代币到meme基金钱包
	sysGroup.POST("/meme/claimToMemeVault", controller.ClaimToMemeVault) //

	//sysGroup.POST("/wallet/createBatch", controller.CreateWalletByUserNoWithNoAuth) //

	// ido 交易验证入库
	sysGroup.POST("/wallet/idoVerify", controller.IdoVerify) //
	// ido 交易验证入库
	sysGroup.POST("/wallet/idoQuery", controller.IdoQuery) //

	// airdrop 查询
	sysGroup.POST("/wallet/airdropQuery", controller.AirdropQuery) //
	// airdrop 查询
	sysGroup.POST("/wallet/airdropPage", controller.AirdropPage) //

}
