package unpack

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Ackites/KillWxapkg/internal/enum"

	"github.com/Ackites/KillWxapkg/internal/config"

	"github.com/Ackites/KillWxapkg/internal/util"
	"github.com/dop251/goja"
	"golang.org/x/net/html"
)

// XssParser 结构体定义
type XssParser struct {
	OutputDir string
}

// 相对路径转换
func makeRelativePath(base, target string) (string, error) {
	relPath, err := filepath.Rel(filepath.Dir(base), target)
	if err != nil {
		return "", err
	}
	// 转换为 Unix 风格的路径以保持一致性
	return filepath.ToSlash(relPath), nil
}

func handleEl(el interface{}, k string) string {
	if elArr, ok := el.([]interface{}); ok {
		if len(elArr) == 1 && elArr[0].(int64) == 1 {
			return ""
		}

		switch elArr[0].(int64) {
		case 0:
			return fmt.Sprintf("%vrpx", elArr[1])
		case 2:
			_el := elArr[1]
			var path string

			switch v := _el.(type) {
			case int64:
				//path = makeCStyleName(v)
				return ""
			case string:
				path = v
			case []interface{}:
				return handleEl(_el, k)
			default:
				log.Printf("Unprocessed element found: %v\n", elArr)
				return ""
			}

			if path == "" {
				return ""
			}

			target, err := makeRelativePath(k, path)
			if err != nil {
				log.Printf("Error making relative path: %v\n", err)
				return ""
			}
			if target != "" {
				return fmt.Sprintf(`@import "%s";`+"\n", target)
			}

			return ""
		default:
			log.Printf("Unprocessed data found: %v\n", elArr)
			return ""
		}
	}

	return fmt.Sprintf("%v", el)
}

func styleConversion(path string, content []interface{}) string {
	var newData strings.Builder
	for _, el := range content {
		newData.WriteString(handleEl(el, path))
	}

	return newData.String()
}

func matchScripts(code string) string {
	doc, err := html.Parse(strings.NewReader(code))
	if err != nil {
		log.Printf("Error parsing HTML: %v\n", err)
		return ""
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
	return scriptBuilder.String()
}

func getCss(mainCode string) string {
	setRe := regexp.MustCompile(`setCssToHead\(([\s\S]*?)\.wxss"\s*\}\)`)
	comRe := regexp.MustCompile(`__COMMON_STYLESHEETS__\['([^']*\.wxss)'\]\s*=\s*\[(.*?)\];`)

	var scriptBuilder strings.Builder

	// 查找所有匹配的子字符串
	matches := setRe.FindAllString(mainCode, -1)
	if len(matches) > 0 {
		for _, match := range matches {
			res := match[strings.LastIndex(match, "setCssToHead("):]
			scriptBuilder.WriteString(res)
			scriptBuilder.WriteString(";\n")
		}
	}
	matches = comRe.FindAllString(mainCode, -1)
	if len(matches) > 0 {
		for _, match := range matches {
			scriptBuilder.WriteString(match)
			scriptBuilder.WriteString(";\n")
		}
	}
	return scriptBuilder.String()
}

// 运行 JavaScript 代码
func runVM(name, code string, results map[string]string) {
	vm := goja.New()

	// 设置 __COMMON_STYLESHEETS__
	commonStylesheets := make(map[string][]interface{})
	err := vm.Set("__COMMON_STYLESHEETS__", commonStylesheets)
	if err != nil {
		log.Printf("Error setting __COMMON_STYLESHEETS__: %v\n", err)
		return
	}

	// 设置 setCssToHead 函数
	err = vm.Set("setCssToHead", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 || (len(call.Arguments) == 1 && len(call.Argument(0).Export().([]interface{})) == 0) {
			return goja.Undefined()
		}

		args := call.Arguments

		if len(call.Arguments) == 3 {
			args = append(args[:1], args[2:]...)
		}

		var sources = args[0].Export().([]interface{})
		var path = args[1].Export().(map[string]interface{})["path"].(string)

		result := styleConversion(path, sources)
		results[path] += result
		return goja.Undefined()
	})
	if err != nil {
		log.Printf("Error setting setCssToHead: %v\n", err)
		return
	}

	// 运行 JavaScript 代码
	_, err = vm.RunString(code)
	if err != nil {
		log.Printf("Error running JavaScript code: %v\n", err)
		return
	}

	// 处理 __COMMON_STYLESHEETS__
	for path, sources := range commonStylesheets {
		result := styleConversion(path, sources)
		results[path] += result
	}
}

// Parse 处理 WXSS 文件
func (p *XssParser) Parse(option config.WxapkgInfo) error {
	saveDir := option.SourcePath
	if isSubpackage(&option) {
		saveDir = p.OutputDir
	}

	// 创建文件删除管理器
	manager := config.NewFileDeletionManager()

	var runList = make(map[string]string)
	var result = make(map[string]string)

	// 预运行，读取所有相关文件
	preRun := func(dir, mainCode string, files []string, cb func()) {
		mainCode = getCss(mainCode)
		if isSubpackage(&option) {
			runList[filepath.Join(option.SourcePath, "./app.wxss")] = mainCode
		} else {
			runList[filepath.Join(dir, "./app.wxss")] = mainCode
		}

		for _, name := range files {
			code, err := os.ReadFile(name)
			if err != nil {
				log.Printf("Error reading file: %v\n", err)
				continue
			}
			codeStr := matchScripts(string(code))

			// 正则表达式：匹配 setCssToHead 函数调用及其内容
			re := regexp.MustCompile(`setCssToHead\(([\s\S]*?\.wxss"[\s\S]*?)\s*\)`)

			// 查找匹配的子字符串
			match := re.FindStringSubmatch(codeStr)
			if len(match) > 0 {
				runList[name] = match[0]
			}
		}

		cb()
	}

	// 一次性运行所有 JavaScript 代码
	runOnce := func() {
		for name, code := range runList {
			runVM(name, code, result)
		}
	}

	// 扫描目录中的所有 HTML 文件
	scanHtml(saveDir, manager, func(files []string) {
		var frameFile = option.Option.ViewSource

		code, err := os.ReadFile(frameFile)
		if err != nil {
			log.Printf("Error reading file: %v\n", err)
			return
		}

		codeStr := string(code)
		scriptCode := codeStr

		if strings.HasSuffix(frameFile, ".html") {
			scriptCode = matchScripts(codeStr)
		}

		preRun(saveDir, scriptCode, files, func() {
			runOnce()
			for name, content := range result {
				name = filepath.Join(saveDir, changeExt(name, ".wxss"))
				err = save(name, []byte(util.TransformCSS(content)))
				if err != nil {
					log.Printf("Error saving file: %v\n", err)
				}
				log.Printf("Saved file: %s\n", name)
			}
		})
	})

	return nil
}

// scanHtml 扫描目录中的Html文件并返回文件列表
func scanHtml(dir string, manager *config.FileDeletionManager, cb func([]string)) {
	var files []string
	// 删除相关的JS文件
	suffixes := []string{".appservice.js", ".common.js", ".webview.js"}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".html") {
			if info.Name() != enum.PageFrameHtml {
				files = append(files, path)
				for _, suffix := range suffixes {
					jsFile := strings.TrimSuffix(path, ".html") + suffix
					manager.AddFile(jsFile)
				}
				manager.AddFile(path)
			}
		}
		return nil
	})
	if err != nil {
		return
	}
	cb(files)
}
