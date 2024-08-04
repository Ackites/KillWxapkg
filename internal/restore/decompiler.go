package restore

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/Ackites/KillWxapkg/internal/unpack"

	"github.com/Ackites/KillWxapkg/internal/util"

	"github.com/Ackites/KillWxapkg/internal/config"
	"github.com/Ackites/KillWxapkg/internal/enum"
)

type WxapkgDecompiler struct {
}

func isParserV1(wxapkg *config.WxapkgInfo) bool {
	switch wxapkg.WxapkgType {
	case enum.App_V1, enum.App_V2, enum.APP_SUBPACKAGE_V1:
		return true
	default:
		return false
	}
}

func isParserV2(wxapkg *config.WxapkgInfo) bool {
	switch wxapkg.WxapkgType {
	case enum.App_V3, enum.App_V4, enum.APP_SUBPACKAGE_V2, enum.APP_PLUGIN_V1:
		return true
	default:
		return false
	}
}

// IsMainPackage 是否主包
func IsMainPackage(wxapkg *config.WxapkgInfo) bool {
	switch wxapkg.WxapkgType {
	case enum.App_V1, enum.App_V2, enum.App_V3, enum.App_V4, enum.GAME:
		return true
	default:
		return false
	}
}

// IsSubpackage 是否分包
func IsSubpackage(wxapkg *config.WxapkgInfo) bool {
	switch wxapkg.WxapkgType {
	case enum.APP_SUBPACKAGE_V1, enum.APP_SUBPACKAGE_V2, enum.GAME_SUBPACKAGE:
		return true
	default:
		return false
	}
}

// 是否小程序插件
func isAppPlugin(wxapkg *config.WxapkgInfo) bool {
	switch wxapkg.WxapkgType {
	case enum.APP_PLUGIN_V1:
		return true
	default:
		return false
	}

}

// 是否游戏插件
func isGamePlugin(wxapkg *config.WxapkgInfo) bool {
	switch wxapkg.WxapkgType {
	case enum.GAME_PLUGIN:
		return true
	default:
		return false
	}
}

// 是否插件
func isPlugin(wxapkg *config.WxapkgInfo) bool {
	return isAppPlugin(wxapkg) || isGamePlugin(wxapkg)
}

// OutputDir 输出目录
var OutputDir string

func (d *WxapkgDecompiler) Decompile(outputDir string) {
	// 设置输出目录
	OutputDir = outputDir

	wxapkgManager := config.GetWxapkgManager()
	for _, wxapkg := range wxapkgManager.Packages {
		log.Println(wxapkg.WxapkgType)
		switch wxapkg.WxapkgType {
		case enum.App_V1, enum.App_V4:
			wxapkg.Option = &config.WxapkgOption{
				ViewSource:   filepath.Join(wxapkg.SourcePath, enum.PageFrameHtml),
				SetAppConfig: true,
			}
			setApp(wxapkg)
		case enum.App_V2, enum.App_V3:
			wxapkg.Option = &config.WxapkgOption{
				SetAppConfig: true,
			}
			setApp(wxapkg)
		case enum.APP_SUBPACKAGE_V1, enum.APP_SUBPACKAGE_V2:
			wxapkg.Option = &config.WxapkgOption{
				ViewSource:   filepath.Join(wxapkg.SourcePath, enum.Page_Frame),
				SetAppConfig: false,
			}
			setApp(wxapkg)
		case enum.APP_PLUGIN_V1:
			wxapkg.Option = &config.WxapkgOption{
				ViewSource:    filepath.Join(wxapkg.SourcePath, enum.PageFrame),
				ServiceSource: filepath.Join(wxapkg.SourcePath, enum.AppService),
				SetAppConfig:  false,
			}
			setApp(wxapkg)
		case enum.GAME:
		case enum.GAME_SUBPACKAGE:
		case enum.GAME_PLUGIN:
		}
	}
}

func setApp(wxapkg *config.WxapkgInfo) {
	// 如果未解压，则不进行解析
	if !wxapkg.IsExtracted {
		return
	}

	if wxapkg.Option == nil {
		wxapkg.Option = &config.WxapkgOption{}
	}

	if wxapkg.Option.ServiceSource == "" {
		wxapkg.Option.ServiceSource = filepath.Join(wxapkg.SourcePath, enum.App_Service)
	}

	if wxapkg.Option.ViewSource == "" {
		wxapkg.Option.ViewSource = filepath.Join(wxapkg.SourcePath, enum.AppWxss)
	}

	wccVersion := util.GetWccVersion(wxapkg.Option.ViewSource)
	if wccVersion != "" {
		log.Printf(fmt.Sprintf("The package %s wcc version is: [%s]", wxapkg.SourcePath, wccVersion))
	}

	if wxapkg.Option.SetAppConfig {
		if wxapkg.Option.AppConfigSource == "" {
			wxapkg.Option.AppConfigSource = filepath.Join(wxapkg.SourcePath, enum.App_Config)
		}
		wxapkg.Parsers = append(wxapkg.Parsers, &unpack.ConfigParser{})
	}

	wxapkg.Parsers = append(wxapkg.Parsers, &unpack.JavaScriptParser{OutputDir: OutputDir})
	wxapkg.Parsers = append(wxapkg.Parsers, &unpack.XssParser{OutputDir: OutputDir})
	if isParserV1(wxapkg) {
		wxapkg.Parsers = append(wxapkg.Parsers, &unpack.XmlParser{OutputDir: OutputDir, Version: "v1"})
	} else if isParserV2(wxapkg) {
		wxapkg.Parsers = append(wxapkg.Parsers, &unpack.XmlParser{OutputDir: OutputDir, Version: "v2"})
	}

	// 清除无用文件
	cleanApp(wxapkg.SourcePath)
}

func cleanApp(path string) {
	// 创建文件删除管理器
	manager := config.NewFileDeletionManager()

	// 删除相关的JS文件, unlinks
	unlinks := []string{
		//".appservice.js",
		"appservice.js",
		"app-config.json",
		"app-service.js",
		"app-wxss.js",
		"appservice.app.js",
		"common.app.js",
		"page-frame.js",
		"page-frame.html",
		"pageframe.js",
		"webview.app.js",
		"subContext.js",
		"plugin.js",
	}

	for _, unlink := range unlinks {
		manager.AddFile(filepath.Join(path, unlink))
	}
}
