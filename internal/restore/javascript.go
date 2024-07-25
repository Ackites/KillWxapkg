package restore

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Ackites/KillWxapkg/internal/unpack"
)

// ProcessJavaScriptFiles 分割JavaScript文件
func ProcessJavaScriptFiles(configFile, outputDir string) error {
	content, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config struct {
		SubPackages []unpack.SubPackage `json:"subPackages"`
	}
	err = json.Unmarshal(content, &config)
	if err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	err = unpack.ProcessJavaScriptFiles(outputDir, config)
	if err != nil {
		return fmt.Errorf("处理JavaScript文件失败: %v", err)
	}

	return nil
}
