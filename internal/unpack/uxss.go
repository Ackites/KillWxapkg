package unpack

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Ackites/KillWxapkg/utils"

	"github.com/dop251/goja"
	"golang.org/x/net/html"
)

// GwxCfg 结构体定义
type GwxCfg struct{}

// GWX 空方法
func (g *GwxCfg) GWX() {}

// CSS 重建函数
func cssRebuild(pureData map[string]interface{}, importCnt map[string]int, actualPure map[string]string, blockCss []string, cssFile string, result map[string]string, commonStyle map[string]interface{}, onlyTest *bool) func(data interface{}) {
	// 统计导入的 CSS 文件
	var statistic func(data interface{})
	statistic = func(data interface{}) {
		addStat := func(id string) {
			if _, exists := importCnt[id]; !exists {
				importCnt[id] = 1
				statistic(pureData[id])
			} else {
				importCnt[id]++
			}
		}

		switch v := data.(type) {
		case float64:
			addStat(fmt.Sprintf("%v", v))
		case string:
			addStat(v)
		case []interface{}:
			for _, content := range v {
				if contentArr, ok := content.([]interface{}); ok && int(contentArr[0].(float64)) == 2 {
					addStat(fmt.Sprintf("%v", contentArr[1]))
				}
			}
		}
	}

	// 生成 CSS 样式
	var makeup func(data interface{}) string
	makeup = func(data interface{}) string {
		log.Printf("Processing data: %v\n", data)
		isPure := false
		switch data.(type) {
		case string:
			isPure = true
		}
		if *onlyTest {
			statistic(data)
			if !isPure {
				dataArr := data.([]interface{})
				if len(dataArr) == 1 && int(dataArr[0].([]interface{})[0].(float64)) == 2 {
					data = dataArr[0].([]interface{})[1]
				} else {
					return ""
				}
			}
			if actualPure[data.(string)] == "" && !contains(blockCss, changeExt(toDir2(cssFile, ""), "")) {
				actualPure[data.(string)] = cssFile
			}
			return ""
		}

		var res strings.Builder
		attach := ""
		if isPure && actualPure[data.(string)] != cssFile {
			if actualPure[data.(string)] != "" {
				return fmt.Sprintf("@import \"%s.wxss\";\n", toDir2(actualPure[data.(string)], cssFile))
			}
			res.WriteString(fmt.Sprintf("/*! Import by _C[%s], whose real path we cannot found. */", data.(string)))
			attach = "/*! Import end */"
		}

		exactData := data
		if isPure {
			exactData = pureData[data.(string)]
		}

		if styleData, ok := commonStyle[data.(string)]; ok {
			if styleArray, ok := styleData.([]interface{}); ok {
				var fileStyle strings.Builder
				for _, content := range styleArray {
					if contentStr, ok := content.(string); ok && contentStr != "1" {
						fileStyle.WriteString(contentStr)
					} else if contentArr, ok := content.([]interface{}); ok && len(contentArr) != 1 {
						fileStyle.WriteString(fmt.Sprintf("%vrpx", contentArr[1]))
					}
				}
				return fileStyle.String()
			}
		} else {
			if dataArr, ok := exactData.([]interface{}); ok {
				for _, content := range dataArr {
					if contentArr, ok := content.([]interface{}); ok {
						switch int(contentArr[0].(float64)) {
						case 0: // rpx
							res.WriteString(fmt.Sprintf("%vrpx", contentArr[1]))
						case 1: // add suffix, ignore it for restoring correct!
						case 2: // import
							res.WriteString(makeup(contentArr[1]))
						}
					} else {
						res.WriteString(content.(string))
					}
				}
			}
		}
		return res.String() + attach
	}

	// 返回处理函数
	return func(data interface{}) {
		log.Printf("Processing CSS file: %s\n", cssFile)
		if result[cssFile] == "" {
			result[cssFile] = ""
		}
		result[cssFile] += makeup(data)
	}
}

// 运行 JavaScript 代码
func runVM(name string, code string, pureData map[string]interface{}, importCnt map[string]int, actualPure map[string]string, blockCss []string, result map[string]string, commonStyle map[string]interface{}, onlyTest *bool) {
	vm := goja.New()
	wxAppCode := make(map[string]func())

	// 添加 console.log
	err := vm.Set("console", map[string]interface{}{
		"log": func(msg string) {
			log.Printf(msg)
		},
	})
	if err != nil {
		log.Printf("Error setting console log: %v\n", err)
		return
	}

	// 设置 setCssToHead 函数
	err = vm.Set("setCssToHead", cssRebuild(pureData, importCnt, actualPure, blockCss, name, result, commonStyle, onlyTest))
	if err != nil {
		log.Printf("Error setting setCssToHead: %v\n", err)
		return
	}

	// 设置 __wxAppCode__
	err = vm.Set("__wxAppCode__", wxAppCode)
	if err != nil {
		log.Printf("Error setting __wxAppCode__: %v\n", err)
		return
	}

	// 运行 JavaScript 代码
	_, err = vm.RunString(code)
	if err != nil {
		log.Printf("Error running JavaScript code: %v\n", err)
		return
	}

	// 调用 wxAppCode 中的函数
	log.Printf("Running wxAppCode...\n")
	for _, fn := range wxAppCode {
		fn()
	}
}

// ProcessXssFiles 处理 WXSS 文件
func ProcessXssFiles(dir string, mainDir string) {
	saveDir := dir
	isSubPkg := mainDir != ""
	if isSubPkg {
		saveDir = mainDir
	}

	var runList = make(map[string]string)
	var pureData = make(map[string]interface{})
	var result = make(map[string]string)
	var actualPure = make(map[string]string)
	var importCnt = make(map[string]int)
	var blockCss []string // 自定义阻止导入的 CSS 文件（无扩展名）
	var commonStyle map[string]interface{}
	var onlyTest = true

	// 预运行，读取所有相关文件
	preRun := func(dir, frameFile, mainCode string, files []string, cb func()) {
		runList[filepath.Join(dir, "./app.wxss")] = mainCode

		for _, name := range files {
			if name != frameFile {
				code, err := os.ReadFile(name)
				if err != nil {
					log.Printf("Error reading file: %v\n", err)
					continue
				}
				codeStr := string(code)
				codeStr = strings.Replace(codeStr, "display:-webkit-box;display:-webkit-flex;", "", -1)
				codeStr = codeStr[:strings.Index(codeStr, "\n")] // 确保只取第一行
				if strings.Contains(codeStr, "setCssToHead(") {
					runList[name] = codeStr[strings.Index(codeStr, "setCssToHead("):]
				}
			}
		}

		cb()
	}

	// 一次性运行所有 JavaScript 代码
	runOnce := func() {
		for name, code := range runList {
			runVM(name, code, pureData, importCnt, actualPure, blockCss, result, commonStyle, &onlyTest)
		}
	}

	// 扫描目录中的所有 HTML 文件
	scanDirByExtTo(dir, ".html", func(files []string) {
		var frameFile string
		if _, err := os.Stat(filepath.Join(dir, "page-frame.html")); err == nil {
			frameFile = filepath.Join(dir, "page-frame.html")
		} else if _, err := os.Stat(filepath.Join(dir, "app-wxss.js")); err == nil {
			frameFile = filepath.Join(dir, "app-wxss.js")
		} else if _, err := os.Stat(filepath.Join(dir, "page-frame.js")); err == nil {
			frameFile = filepath.Join(dir, "page-frame.js")
		} else {
			log.Printf("未找到类似 page-frame 的文件")
			return
		}

		code, err := os.ReadFile(frameFile)
		if err != nil {
			log.Printf("Error reading file: %v\n", err)
			return
		}

		codeStr := string(code)
		codeStr = strings.Replace(codeStr, "display:-webkit-box;display:-webkit-flex;", "", -1)
		scriptCode := codeStr

		if strings.HasSuffix(frameFile, ".html") {
			doc, err := html.Parse(strings.NewReader(codeStr))
			if err != nil {
				log.Printf("Error parsing HTML: %v\n", err)
				return
			}
			var scriptBuilder strings.Builder
			var f func(*html.Node)
			f = func(n *html.Node) {
				if n.Type == html.ElementNode && n.Data == "script" {
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						if c.Type == html.TextNode {
							scriptBuilder.WriteString(c.Data)
						}
					}
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					f(c)
				}
			}
			f(doc)
			scriptCode = scriptBuilder.String()
		}

		window := map[string]interface{}{
			"screen": map[string]interface{}{
				"width":  720,
				"height": 1028,
				"orientation": map[string]interface{}{
					"type": "vertical",
				},
			},
		}
		navigator := map[string]interface{}{
			"userAgent": "iPhone",
		}

		scriptCode = scriptCode[strings.LastIndex(scriptCode, "window.__wcc_version__"):]
		mainCode := fmt.Sprintf(`window=%s;navigator=%s;var __mainPageFrameReady__=window.__mainPageFrameReady__ || function() {};var __WXML_GLOBAL__={entrys:{},defines:{},modules:{},ops:[],wxs_nf_init:undefined,total_ops:0};var __vd_version_info__=__vd_version_info__ || {};%s`,
			toJSON(window), toJSON(navigator), scriptCode)

		// 处理 commonStyles
		if idx := strings.Index(codeStr, "__COMMON_STYLESHEETS__ || {}"); idx != -1 {
			start := idx + 28
			end := strings.Index(codeStr[start:], "var setCssToHead = function(file, _xcInvalid, info)")
			if end != -1 {
				commonStyles := codeStr[start : start+end]
				vm := goja.New()
				_, err := vm.RunString(fmt.Sprintf(";var __COMMON_STYLESHEETS__ = __COMMON_STYLESHEETS__ || {};%s;__COMMON_STYLESHEETS__;", commonStyles))
				if err == nil {
					err = vm.ExportTo(vm.Get("__COMMON_STYLESHEETS__"), &commonStyle)
					if err != nil {
						log.Printf("Error exporting common styles: %v\n", err)
					}
				}
			}
		}

		mainCode = strings.Replace(mainCode, "var setCssToHead = function", "var setCssToHead2 = function", 1)
		codeStr = codeStr[strings.LastIndex(codeStr, "var setCssToHead = function(file, _xcInvalid"):]
		codeStr = strings.Replace(codeStr, "__COMMON_STYLESHEETS__", "[]", 1)
		if idx := strings.Index(codeStr, "_C = "); idx != -1 {
			codeStr = codeStr[strings.LastIndex(codeStr, "var _C = "):]
		} else {
			codeStr = codeStr[strings.LastIndex(codeStr, "var _C= "):]
		}

		codeStr = codeStr[:strings.Index(codeStr, "\n")]

		vm := goja.New()
		_, err = vm.RunString(codeStr + "\n_C")
		if err != nil {
			log.Printf("Error running JavaScript code: %v\n", err)
			return
		}
		err = vm.ExportTo(vm.Get("_C"), &pureData)
		if err != nil {
			log.Printf("Error exporting pure data: %v\n", err)
			return
		}

		preRun(dir, frameFile, mainCode, files, func() {
			runOnce()
			onlyTest = false
			runOnce()

			for name, content := range result {
				name = filepath.Join(saveDir, changeExt(name, ".wxss"))
				err := os.WriteFile(name, []byte(utils.TransformCSS(content)), 0755)
				log.Printf("Save wxss %s done.\n", name)
				if err != nil {
					log.Printf("Error saving file: %v\n", err)
				}
			}
		})
	})
}

// toJSON 将对象转换为 JSON 字符串
func toJSON(obj interface{}) string {
	data, _ := json.Marshal(obj)
	return string(data)
}

// scanDirByExtTo 扫描目录中的文件并返回指定扩展名的文件列表
func scanDirByExtTo(dir, ext string, cb func([]string)) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(info.Name(), ext) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return
	}
	cb(files)
}

// contains 检查切片是否包含特定元素
func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// toDir2 生成目录路径
func toDir2(filePath, frameName string) string {
	dir := filepath.Dir(filePath)
	return filepath.Join(dir, frameName)
}
