package hook

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func Hook() {
	// 检查是否在 Windows 环境中运行
	if runtime.GOOS != "windows" {
		fmt.Println("Not running on Windows. Exiting hook.")
		return
	}

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "KillwxapkgHook")
	if err != nil {
		fmt.Printf("Failed to create temporary directory: %v\n", err)
		return
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			fmt.Printf("Failed to remove temporary directory: %v\n", err)
		}
	}(tempDir) // 确保在程序退出时删除临时目录

	exePath := filepath.Join(tempDir, "win.exe")

	// 将嵌入的 exe 文件写入到临时目录
	err = os.WriteFile(exePath, embeddedExe, 0755)
	if err != nil {
		fmt.Printf("Failed to write embedded exe file: %v\n", err)
		return
	}

	// 执行临时目录中的 exe 文件
	cmd := exec.Command(exePath, "-x")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to execute embedded exe file: %v\n", err)
		return
	}

	// 如果输出是 GBK 编码，进行转换
	decoder := transform.NewReader(strings.NewReader(string(output)), simplifiedchinese.GBK.NewDecoder())
	decodedOutput, err := io.ReadAll(decoder)
	if err != nil {
		fmt.Printf("Failed to decode output: %v\n", err)
		return
	}

	// 打印 exe 文件的输出
	fmt.Printf("%s\n", decodedOutput)
}
