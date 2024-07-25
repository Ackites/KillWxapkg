package restore

import (
	"path/filepath"

	"github.com/Ackites/KillWxapkg/internal/unpack"
)

// ProcessWxssFiles 分割 WXSS 文件
func ProcessWxssFiles(outputDir string, config unpack.AppConfig) {
	// 处理主包的 WXSS 文件
	unpack.ProcessXssFiles(outputDir, "")

	// 处理子包的 WXSS 文件
	for _, subPackage := range config.SubPackages {
		subPackageDir := filepath.Join(outputDir, subPackage.Root)
		unpack.ProcessXssFiles(subPackageDir, outputDir)
	}
}
