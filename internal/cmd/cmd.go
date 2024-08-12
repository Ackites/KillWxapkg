package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Ackites/KillWxapkg/internal/restore"
	"github.com/Ackites/KillWxapkg/internal/util"

	. "github.com/Ackites/KillWxapkg/internal/config"
	"github.com/Ackites/KillWxapkg/internal/decrypt"
	"github.com/Ackites/KillWxapkg/internal/unpack"
)

// ParseInput 解析输入文件
func ParseInput(input, fileExt string) []string {
	var inputFiles []string
	if fileInfo, err := os.Stat(input); err == nil && fileInfo.IsDir() {
		files, err := os.ReadDir(input)
		if err != nil {
			log.Fatalf("读取输入目录失败: %v", err)
		}
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), fileExt) {
				inputFiles = append(inputFiles, filepath.Join(input, file.Name()))
			}
		}
	} else {
		inputFiles = strings.Split(input, ",")
	}

	// 过滤掉不存在的文件
	var validFiles []string
	for _, file := range inputFiles {
		if _, err := os.Stat(file); err == nil {
			validFiles = append(validFiles, file)
		}
	}
	return validFiles
}

// DetermineOutputDir 确定输出目录
func DetermineOutputDir(input, appID string) string {
	var baseDir string

	if fileInfo, err := os.Stat(input); err == nil && fileInfo.IsDir() {
		baseDir = input
	} else {
		baseDir = filepath.Dir(input)
	}

	if appID == "" {
		return filepath.Join(baseDir, "result")
	}

	return filepath.Join(baseDir, appID)
}

// ProcessFile 合并目录
func ProcessFile(inputFile, outputDir, appID string, save bool) error {
	log.Printf("开始处理文件: %s\n", inputFile)

	manager := GetWxapkgManager()

	// 初始化 WxapkgInfo
	info := &WxapkgInfo{
		WxAppId:     appID,
		IsExtracted: false,
	}

	// 确定解密后的文件路径
	decryptedFilePath := filepath.Join(outputDir, filepath.Base(inputFile))

	// 解密
	decryptedData, err := decrypt.DecryptWxapkg(inputFile, appID)
	if err != nil {
		return fmt.Errorf("解密失败: %v", err)
	}

	// 保存解密后的文件
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	// 是否保存解密后的文件
	if save {
		err = os.WriteFile(decryptedFilePath, decryptedData, 0755)
		if err != nil {
			return fmt.Errorf("保存解密文件失败: %v", err)
		}

		log.Printf("文件解密并保存到: %s\n", decryptedFilePath)
	}

	// 解包到临时目录
	tempDir, err := os.MkdirTemp("", "wxapkg")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			log.Printf("删除临时目录 %s 失败: %v\n", path, err)
		}
	}(tempDir)

	// 包文件列表
	var filelist []string

	filelist, err = unpack.UnpackWxapkg(decryptedData, tempDir)
	if err != nil {
		return fmt.Errorf("解包失败: %v", err)
	}

	// 设置解包状态
	info.IsExtracted = true

	// 合并解包后的内容到输出目录
	err = mergeDirs(tempDir, outputDir)
	if err != nil {
		return fmt.Errorf("合并目录失败: %v", err)
	}

	info.WxapkgType = util.GetWxapkgType(filelist)

	if restore.IsMainPackage(info) {
		info.SourcePath = outputDir
	} else if restore.IsSubpackage(info) {
		info.SourcePath = filelist[0]
	}

	// 将包信息添加到管理器中
	manager.AddPackage(info.SourcePath, info)

	return nil
}

// mergeDirs 合并目录
func mergeDirs(srcDir, dstDir string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dstDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func(srcFile *os.File) {
			err := srcFile.Close()
			if err != nil {
				log.Printf("关闭文件 %s 失败: %v\n", srcFile.Name(), err)
			}
		}(srcFile)

		dstFile, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer func(dstFile *os.File) {
			err := dstFile.Close()
			if err != nil {
				log.Printf("关闭文件 %s 失败: %v\n", dstFile.Name(), err)
			}
		}(dstFile)

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}
