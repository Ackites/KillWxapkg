package formatter

import (
	"fmt"
	"strings"

	. "github.com/Ackites/KillWxapkg/internal/config"
)

// Formatter 是一个文件格式化器接口
type Formatter interface {
	Format([]byte) ([]byte, error)
}

// 注册所有格式化器
var formatters = map[string]Formatter{}

// RegisterFormatter 注册文件扩展名对应的格式化器
func RegisterFormatter(ext string, formatter Formatter) {
	formatters[strings.ToLower(ext)] = formatter
}

// GetFormatter 返回文件扩展名对应的格式化器
func GetFormatter(ext string) (Formatter, error) {
	formatter, exists := formatters[strings.ToLower(ext)]
	if !exists {
		return nil, fmt.Errorf("不支持的文件类型: %s", ext)
	}
	configManager := NewSharedConfigManager()
	if pretty, ok := configManager.Get("pretty"); ok {
		if p, o := pretty.(bool); o {
			if !p && ext == ".js" {
				return nil, fmt.Errorf("不进行美化输出")
			}
		}
	}
	return formatter, nil
}
