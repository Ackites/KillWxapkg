package main

import (
	"flag"
	"fmt"

	"github.com/Ackites/KillWxapkg/internal/pack"

	"github.com/Ackites/KillWxapkg/cmd"
	hook2 "github.com/Ackites/KillWxapkg/internal/hook"
)

var (
	appID      string
	input      string
	outputDir  string
	fileExt    string
	restoreDir bool
	pretty     bool
	noClean    bool
	hook       bool
	save       bool
	repack     string
	watch      bool
	sensitive  bool
)

func init() {
	flag.StringVar(&appID, "id", "", "微信小程序的AppID")
	flag.StringVar(&input, "in", "", "输入文件路径（多个文件用逗号分隔）或输入目录路径")
	flag.StringVar(&outputDir, "out", "", "输出目录路径（如果未指定，则默认保存到输入目录下以AppID命名的文件夹）")
	flag.StringVar(&fileExt, "ext", ".wxapkg", "处理的文件后缀")
	flag.BoolVar(&restoreDir, "restore", false, "是否还原工程目录结构")
	flag.BoolVar(&pretty, "pretty", false, "是否美化输出")
	flag.BoolVar(&noClean, "noClean", false, "是否清理中间文件")
	flag.BoolVar(&hook, "hook", false, "是否开启动态调试")
	flag.BoolVar(&save, "save", false, "是否保存解密后的文件")
	flag.StringVar(&repack, "repack", "", "重新打包wxapkg文件")
	flag.BoolVar(&watch, "watch", false, "是否监听将要打包的文件夹，并自动打包")
	flag.BoolVar(&sensitive, "sensitive", false, "是否获取敏感数据")
}

func main() {
	// 解析命令行参数
	flag.Parse()

	banner := `
 _   __ _ _ _  __      __                 _         
| | / /(_) | | \ \    / /                | |        
| |/ /  _| | |  \ \  / /   __  ____ _ ___| | ____ _ 
|    \ | | | |   \ \/ /   / / / / _  / __| |/ /  ' \
| |\  \| | | |    \  /   / /_/ / (_| \__ \   <| | | |
\_| \_/_|_|_|     \/    \__,_|\__,_|___/_|\_\_| |_|
                                                    
             Wxapkg Decompiler Tool v2.4.0
    `
	fmt.Println(banner)

	// 动态调试
	if hook {
		hook2.Hook()
		return
	}

	// 重新打包
	if repack != "" {
		pack.Repack(repack, watch, outputDir)
		return
	}

	if appID == "" || input == "" {
		fmt.Println("使用方法: program -id=<AppID> -in=<输入文件1,输入文件2> 或 -in=<输入目录> -out=<输出目录> [-ext=<文件后缀>] [-restore] [-pretty] [-noClean] [-hook] [-save] [-repack=<输入目录>] [-watch] [-sensitive]")
		flag.PrintDefaults()
		fmt.Println()
		return
	}

	// 执行命令
	cmd.Execute(appID, input, outputDir, fileExt, restoreDir, pretty, noClean, save, sensitive)
}
