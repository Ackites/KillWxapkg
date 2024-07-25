package restore

import (
	"log"
	"path/filepath"

	"github.com/Ackites/KillWxapkg/internal/config"
	"github.com/Ackites/KillWxapkg/internal/unpack"
)

// ProjectStructure 是否还原工程目录结构
func ProjectStructure(outputDir string, restoreDir bool) {
	if !restoreDir {
		return
	}

	// 创建文件删除管理器
	manager := config.NewFileDeletionManager()

	// 配置文件还原
	configFile := filepath.Join(outputDir, "app-config.json")
	err := unpack.ProcessConfigFiles(configFile)
	if err != nil {
		log.Printf("还原工程目录结构失败: %v\n", err)
	} else {
		manager.AddFile(configFile)
	}

	// JavaScript文件还原
	err = ProcessJavaScriptFiles(configFile, outputDir)
	if err != nil {
		log.Printf("处理JavaScript文件失败: %v\n", err)
	}

	// WXSS文件还原
	//var config unpack.AppConfig
	//content, err := os.ReadFile(configFile)
	//if err == nil {
	//	_ = json.Unmarshal(content, &config)
	//}
	//ProcessWxssFiles(outputDir, config)

	// 执行删除文件操作
	manager.DeleteFiles()
}
