package restore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/Ackites/KillWxapkg/internal/enum"

	"github.com/Ackites/KillWxapkg/internal/config"
	"github.com/Ackites/KillWxapkg/internal/unpack"
)

// fixSubpackageDir 修正子包目录
func fixSubpackageDir(wxapkg *config.WxapkgInfo, outputDir string) string {
	var e struct {
		SubPackages []unpack.SubPackage `json:"subPackages"`
	}
	content, _ := os.ReadFile(filepath.Join(outputDir, enum.App_Config))
	_ = json.Unmarshal(content, &e)

	for _, subPackage := range e.SubPackages {
		root := subPackage.Root
		if !strings.HasPrefix(root, "/") {
			root = "/" + root
		}
		if strings.HasPrefix(wxapkg.SourcePath, root) {
			return filepath.Join(outputDir, root)
		}
	}

	return ""
}

// ProjectStructure 是否还原工程目录结构
func ProjectStructure(outputDir string, restoreDir bool) {
	if !restoreDir {
		return
	}

	configManager := config.NewSharedConfigManager()

	// 创建文件删除管理器
	manager := config.NewFileDeletionManager()

	defer func() {
		if noClean, ok := configManager.Get("noClean"); ok {
			if !noClean.(bool) {
				// 执行删除文件操作
				manager.DeleteFiles()
			}
		}
	}()

	// 包管理器
	wxakpgManager := config.GetWxapkgManager()

	// 修正子包目录
	for _, wxapkg := range wxakpgManager.Packages {
		if IsSubpackage(wxapkg) {
			wxapkg.SourcePath = fixSubpackageDir(wxapkg, outputDir)
		}
	}

	// 反编译
	decompiler := new(WxapkgDecompiler)
	// 执行反编译操作
	decompiler.Decompile(outputDir)

	// 创建命令执行器, 执行解析器
	executor := NewCommandExecutor(wxakpgManager)
	executor.ExecuteAll()
}
