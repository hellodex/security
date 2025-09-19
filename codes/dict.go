package codes

const (
	CODE_SUCCESS              = 0
	CODE_ERR_METHOD_UNSUPPORT = 1
	CODE_ERR_REQFORMAT        = 2
	CODE_ERR_APPID_INVALID    = 3
	CODE_ERR_SIGMETHOD_UNSUPP = 4
	CODE_ERR_AUTHTOKEN_FAIL   = 5
	CODE_ERR_REQ_EXPIRED      = 6
	CODE_ERR_EXIST_OBJ        = 7
	CODE_ERR_BAT_PARAMS       = 8
	CODES_ERR_OBJ_NOT_FOUND   = 9
	CODES_ERR_SIG_COMMON      = 10
	CODES_ERR_CONFIG          = 11
	CODES_ERR_PARA_EMPTY      = 12
	CODES_ERR_TX              = 13
	CODES_ERR_INFO            = 14 // msg错误信息 在前端和移动端直接显示

	CODE_ERR_UNKNOWN = 900
	CODE_ERR_INVALID = 400

	CODE_ERR_LAN           = 901
	CODE_ERR_CHAR_BASPARAM = 100
	CODE_ERR_CHAR_UNKNOWN  = 101
	CODE_ERR_CHAR_NOTFOUND = 102

	CODE_ERR_CHAR_PARAM   = 104
	CODE_ERR_CHARBACK_MAX = 105
	CODE_ERR_CHAR_EXIST   = 106

	CODE_ERR_GPT_COMPLETE   = 201
	CODE_ERR_GPT_STREAM     = 202
	CODE_ERR_GPT_STREAM_EOF = 203

	CODE_ERR             = 400
	CODE_ERR_102         = 102
	CODE_ERR_103         = 103
	CODE_ERR_AUTH_FAIL   = 4011
	CODE_ERR_VERIFY_FAIL = 4013
	/*
	   E4011(4011,  "无登录信息,请登录后重试" ,"" ),
	   E4013(4013,  "验证码错误" ,"" ),
	   E4014(4014,  "账号或密码错误" ,"" ),
	   E4015(4015,  "密码强度不够" ,"" ),
	   E4016(4016,  "创建用户失败" ,"" ),
	   E4017(4017,  "邀请码错误" ,"" ),
	   E4018(4018,  "账户已注册,请登录" ,"" ),
	   E4019(4019,  "账号已被禁用" ,"" ),
	   E4020(4020,  "账号已过期" ,"" ),
	*/
	CODE_ERR_4011    = 4011
	CODE_ERR_4013    = 4013
	CODE_ERR_4014    = 4014
	CODE_ERR_4015    = 4015
	CODE_ERR_4016    = 4016
	CODE_ERR_4017    = 4017
	CODE_ERR_4018    = 4018
	CODE_ERR_4019    = 4019
	CODE_ERR_4020    = 4020
	CODE_SUCCESS_200 = 200
)

//E400(400,  "内部服务器错误" ,"" ),
//E404(404,  "内部服务器错误" ,"" ),
//E429(429,  "请求太频繁" ,"" ),
//E439(439,  "余额不足" ,"" ),
//E440(440,  "代币余额不足" ,"" ),
//
//E4011(4011,  "无登录信息,请登录后重试" ,"" ),
//E4013(4013,  "验证码错误" ,"" ),
//E4014(4014,  "账号或密码错误" ,"" ),
//E4015(4015,  "密码强度不够" ,"" ),
//E4016(4016,  "创建用户失败" ,"" ),
//E4017(4017,  "邀请码错误" ,"" ),
//
//;
//
//
//private final int code;
//private final String Msg;
//private final String i18n;
