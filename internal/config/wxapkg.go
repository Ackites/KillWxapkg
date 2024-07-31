package config

import (
	"sync"

	"github.com/Ackites/KillWxapkg/internal/enum"
)

// Parser 接口
type Parser interface {
	Parse(option WxapkgInfo) error
}

// WxapkgOption 微信小程序解包选项
type WxapkgOption struct {
	ViewSource      string
	AppConfigSource string
	ServiceSource   string
	SetAppConfig    bool
}

// WxapkgInfo 保存包的信息
type WxapkgInfo struct {
	WxAppId     string
	WxapkgType  enum.WxapkgType
	SourcePath  string
	IsExtracted bool
	Option      *WxapkgOption
	Parsers     []Parser // 添加解析器列表
}

// WxapkgManager 管理多个微信小程序包
type WxapkgManager struct {
	Packages map[string]*WxapkgInfo
}

var managerInstance *WxapkgManager
var wxapkgOnce sync.Once

// GetWxapkgManager 获取单例的 WxapkgManager 实例
func GetWxapkgManager() *WxapkgManager {
	wxapkgOnce.Do(func() {
		managerInstance = &WxapkgManager{
			Packages: make(map[string]*WxapkgInfo),
		}
	})
	return managerInstance
}

// AddPackage 添加包信息
func (manager *WxapkgManager) AddPackage(id string, info *WxapkgInfo) {
	manager.Packages[id] = info
}

// GetPackage 获取包信息
func (manager *WxapkgManager) GetPackage(id string) (*WxapkgInfo, bool) {
	info, exists := manager.Packages[id]
	return info, exists
}
