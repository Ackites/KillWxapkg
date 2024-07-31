package util

import (
	"path/filepath"
	"strings"

	. "github.com/Ackites/KillWxapkg/internal/enum"
)

// GetWxapkgType 根据文件列表判断微信小程序包的类型
func GetWxapkgType(fileList []string) WxapkgType {
	allFilesStartWithWA := true
	for _, filename := range fileList {
		if !strings.HasPrefix(filepath.Base(filename), "WA") {
			allFilesStartWithWA = false
			break
		}
	}

	if allFilesStartWithWA {
		return FRAMEWORK
	}

	if containsFile(fileList, PageFrameHtml) {
		if containsFile(fileList, CommonApp) {
			return App_V4
		}
		return App_V1
	}

	if containsFile(fileList, CommonApp) {
		if containsFile(fileList, AppWxss) {
			return App_V3
		}
		return APP_SUBPACKAGE_V2
	}

	if containsFile(fileList, Page_Frame) {
		if containsFile(fileList, AppWxss) {
			return App_V2
		}
		return APP_SUBPACKAGE_V1
	}

	if containsFile(fileList, Game) {
		if containsFile(fileList, App_Config) {
			return GAME
		}
		return GAME_SUBPACKAGE
	}

	if containsFile(fileList, PluginJson) {
		if containsFile(fileList, AppService) {
			return APP_PLUGIN_V1
		}
		if containsFile(fileList, Plugin) {
			return GAME_PLUGIN
		}
	}

	return ""
}

// containsFile 检查切片中是否包含特定文件名
func containsFile(slice []string, filename string) bool {
	for _, element := range slice {
		if filepath.Base(element) == filename {
			return true
		}
	}
	return false
}
