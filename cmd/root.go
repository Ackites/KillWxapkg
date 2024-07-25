package cmd

import (
	"flag"
	"fmt"
	"log"
	"sync"

	. "github.com/Ackites/KillWxapkg/internal/cmd"
	. "github.com/Ackites/KillWxapkg/internal/config"
	"github.com/Ackites/KillWxapkg/internal/restore"
)

func Execute(appID, input, outputDir, fileExt string, restoreDir bool, pretty bool) {
	if appID == "" || input == "" {
		fmt.Println("使用方法: program -id=<AppID> -in=<输入文件1,输入文件2> 或 -in=<输入目录> -out=<输出目录> [-ext=<文件后缀>] [-restore]")
		flag.PrintDefaults()
		return
	}

	// 存储配置
	configManager := NewSharedConfigManager()
	configManager.Set("appID", appID)
	configManager.Set("input", input)
	configManager.Set("outputDir", outputDir)
	configManager.Set("fileExt", fileExt)
	configManager.Set("restoreDir", restoreDir)
	configManager.Set("pretty", pretty)

	inputFiles := ParseInput(input, fileExt)

	if len(inputFiles) == 0 {
		log.Println("未找到任何文件")
		return
	}

	// 确定输出目录
	if outputDir == "" {
		outputDir = DetermineOutputDir(input, appID)
	}

	var wg sync.WaitGroup
	for _, inputFile := range inputFiles {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			err := ProcessFile(file, outputDir, appID)
			if err != nil {
				log.Printf("处理文件 %s 时出错: %v\n", file, err)
			} else {
				log.Printf("成功处理文件: %s\n", file)
			}
		}(inputFile)
	}
	wg.Wait()

	// 还原工程目录结构
	restore.ProjectStructure(outputDir, restoreDir)
}
