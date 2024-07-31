package enum

// 定义微信小程序包的文件名常量
const (
	App_Config    = "app-config.json" // 应用配置文件
	App_Service   = "app-service.js"  // 应用服务文件
	PageFrameHtml = "page-frame.html" // 页面框架HTML文件
	AppWxss       = "app-wxss.js"     // 应用样式文件
	CommonApp     = "common.app.js"   // 通用应用文件
	AppJson       = "app.json"        // 应用JSON文件
	Workers       = "workers.js"      // 工作者脚本文件
	Page_Frame    = "page-frame.js"   // 页面框架JS文件
	AppService    = "appservice.js"   // 应用服务文件
	PageFrame     = "pageframe.js"    // 页面框架JS文件
	Game          = "game.js"         // 游戏脚本文件
	GameJson      = "game.json"       // 游戏JSON文件
	SubContext    = "subContext.js"   // 子上下文脚本文件
	Plugin        = "plugin.js"       // 插件脚本文件
	PluginJson    = "plugin.json"     // 插件JSON文件
)

// WxapkgType 定义微信小程序包的类型
type WxapkgType string

// 定义微信小程序包的类型常量
const (
	App_V1 WxapkgType = "APP_V1" // 应用类型 V1
	App_V2 WxapkgType = "APP_V2" // 应用类型 V2
	App_V3 WxapkgType = "APP_V3" // 应用类型 V3
	App_V4 WxapkgType = "APP_V4" // 应用类型 V4

	APP_SUBPACKAGE_V1 WxapkgType = "APP_SUBPACKAGE_V1" // 应用子包类型 V1
	APP_SUBPACKAGE_V2 WxapkgType = "APP_SUBPACKAGE_V2" // 应用子包类型 V2

	APP_PLUGIN_V1 WxapkgType = "APP_PLUGIN_V1" // 应用插件类型 V1

	GAME            WxapkgType = "GAME"            // 游戏类型
	GAME_SUBPACKAGE WxapkgType = "GAME_SUBPACKAGE" // 游戏子包类型
	GAME_PLUGIN     WxapkgType = "GAME_PLUGIN"     // 游戏插件类型

	FRAMEWORK WxapkgType = "FRAMEWORK" // 框架类型
)
