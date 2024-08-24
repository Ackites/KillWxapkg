package unpack

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/Ackites/KillWxapkg/internal/config"
	"github.com/dop251/goja"
)

type XmlParser struct {
	OutputDir string
	// 解析器版本
	Version string
}

// 获取生成函数
func getFuc(code string, gwx map[string]interface{}) {
	re := regexp.MustCompile(`else\s+__wxAppCode__\['([^']+\.wxml)'\]\s*=\s*(\$[^;]+;)`)

	matches := re.FindAllStringSubmatch(code, -1)
	if len(matches) > 0 {
		for _, match := range matches {
			gwx[match[1]] = match[2]
		}
	}
}

// 提取函数名和参数
func extractFuncNameAndArgs(gencode string) (string, []interface{}) {
	re := regexp.MustCompile(`(\$\w+)\s*\(\s*'(.*?)'\s*\)`)
	matches := re.FindStringSubmatch(gencode)
	if len(matches) < 3 {
		return "", nil
	}

	funcName := matches[1]
	arg := matches[2]

	return funcName, []interface{}{arg}
}

// 递归调用函数直到获得非函数结果
func getFinalResult(vm *goja.Runtime, value goja.Value) (goja.Value, error) {
	for value.ExportType().Kind() == reflect.Func {
		fn, ok := goja.AssertFunction(value)
		if !ok {
			return nil, fmt.Errorf("expected function, got %T", value.Export())
		}

		var err error
		value, err = fn(goja.Undefined())
		if err != nil {
			return nil, err
		}
	}
	return value, nil
}

// 生成视图代码
func getDomTree(node interface{}) string {
	// 用于构建 XML 字符串的函数
	var processNodes func(node map[string]interface{}, indentLevel int, isRoot bool) string
	processNodes = func(node map[string]interface{}, indentLevel int, isRoot bool) string {
		var sb strings.Builder

		// 生成缩进
		indent := strings.Repeat("\t", indentLevel)

		// 获取标签名称
		tag, ok := node["tag"].(string)
		if !ok {
			return ""
		}
		tag = strings.TrimPrefix(tag, "wx-") // 去除前缀 wx-

		// 如果是根节点，不添加开始标签
		if !isRoot {
			// 开始标签
			sb.WriteString(indent)
			sb.WriteString("<")
			sb.WriteString(tag)

			// 处理属性
			if attr, ok := node["attr"].(map[string]interface{}); ok {
				for key, value := range attr {
					key = strings.TrimPrefix(key, "$wxs:")
					if strings.HasPrefix(key, "$") {
						continue
					}
					if value == nil {
						sb.WriteString(fmt.Sprintf(" %s=\"\"", key))
					} else {
						sb.WriteString(fmt.Sprintf(" %s=\"%v\"", key, value))
					}
				}
			}

			// 结束标签
			sb.WriteString(">")
		}

		// 处理子节点
		if children, ok := node["children"].([]interface{}); ok {
			if len(children) > 0 && !isRoot {
				sb.WriteString("\n")
			}
			for _, child := range children {
				if childMap, ok := child.(map[string]interface{}); ok {
					sb.WriteString(processNodes(childMap, indentLevel+1, false))
				} else {
					// 如果 children 是字符串且字符串为空，则不换行
					if str, ok := child.(string); ok {
						if str != "" {
							sb.WriteString(strings.Repeat("\t", indentLevel+1))
							sb.WriteString(str + "\n")
						}
					}
				}
			}
		}

		// 结束标签（如果不是根节点）
		if !isRoot {
			sb.WriteString(indent)
			sb.WriteString("</")
			sb.WriteString(tag)
			sb.WriteString(">\n")
		}

		return sb.String()
	}

	// 将根节点转换为 map
	rootNode, ok := node.(map[string]interface{})
	if !ok {
		return ""
	}

	// 生成并返回最终的 XML 字符串，不包括根标签
	return processNodes(rootNode, 0, true)
}

func getXml(path string, scriptCode, gencode string, results chan<- map[string]interface{}, wg *sync.WaitGroup, version string, sem chan struct{}) {
	defer wg.Done()

	// 限制并发数
	sem <- struct{}{}
	// 释放信号量
	defer func() { <-sem }()

	// 提取函数名和参数
	funcName, params := extractFuncNameAndArgs(gencode)
	if funcName == "" {
		log.Printf("Error extracting function name and arguments from gencode: %s\n", gencode)
		return
	}

	vm := goja.New()

	// 包裹 try...catch 语句以捕获 JavaScript 错误
	safeScript := `
	try {
		` + scriptCode + `
	} catch (e) {
		//console.error(e);
	}
	`

	// 定义 console 对象
	console := vm.NewObject()
	_ = console.Set("log", func(call goja.FunctionCall) goja.Value {
		// 使用 call.Arguments 获取传递给 console.log 的参数
		args := call.Arguments
		for _, arg := range args {
			fmt.Println(arg.String())
		}
		return goja.Undefined()
	})
	_ = console.Set("error", func(call goja.FunctionCall) goja.Value {
		args := call.Arguments
		for _, arg := range args {
			fmt.Println("ERROR:", arg.String())
		}
		return goja.Undefined()
	})
	_ = console.Set("warn", func(call goja.FunctionCall) goja.Value {
		args := call.Arguments
		for _, arg := range args {
			fmt.Println("WARN:", arg.String())
		}
		return goja.Undefined()
	})
	_ = vm.Set("console", console)

	// 运行脚本代码，定义所有函数
	_, err := vm.RunString(safeScript)
	if err != nil {
		var gojaErr *goja.Exception
		if errors.As(err, &gojaErr) {
			log.Println("JavaScript error:", gojaErr.String())
		} else {
			log.Println("Error running script:", err)
		}
		return
	}

	// 获取函数对象
	fn, ok := goja.AssertFunction(vm.Get(funcName))
	if !ok {
		log.Printf("Error asserting function for %s\n", funcName)
		return
	}

	// 准备参数列表
	args := make([]goja.Value, len(params))
	for i, param := range params {
		args[i] = vm.ToValue(param)
	}

	// 调用函数并获取结果
	result, err := fn(goja.Undefined(), args...)
	if err != nil {
		log.Printf("Error calling function: %v\n", err)
		return
	}

	// 递归调用函数直到获得非函数结果
	finalResult, err := getFinalResult(vm, result)
	if err != nil {
		log.Printf("Error getting final result: %v\n", err)
		return
	}

	// 保存结果
	results <- map[string]interface{}{path: finalResult.Export()}
}

func (p *XmlParser) Parse(option config.WxapkgInfo) error {
	saveDir := p.OutputDir

	var frameFile = option.Option.ViewSource
	// 存放生成函数代码
	var gwx = make(map[string]interface{})
	results := make(chan map[string]interface{})
	var wg sync.WaitGroup

	// 最大并发数
	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)

	code, err := os.ReadFile(frameFile)
	if err != nil {
		log.Printf("Error reading file: %v\n", err)
		return err
	}

	codeStr := string(code)
	scriptCode := codeStr

	// 防止报错
	patch := `var noCss=true;var window={};var navigator={};navigator.userAgent="iPhone";window.screen={};
document={getElementsByTagName:()=>{}};function define(){};function require(){};
var setCssToHead=function(file,_xcInvalid,info){return ()=>{}};`

	// 如果是 html 文件，提取 script 代码
	if strings.HasSuffix(frameFile, ".html") {
		scriptCode = matchScripts(codeStr)
	}

	scriptCode = strings.Replace(scriptCode, "var setCssToHead =", "var setCssToHead2 =", -1)
	scriptCode = strings.Replace(scriptCode, "var noCss", "var noCss2", -1)
	// 如果是子包
	if isSubpackage(&option) {
		scriptCode = strings.Replace(scriptCode, "$gwx('init', global);", "", 1)
	}

	// 正则匹配生成函数
	getFuc(scriptCode, gwx)

	scriptCode = patch + scriptCode

	// 运行生成函数
	for path, gencode := range gwx {
		wg.Add(1)
		go getXml(path, scriptCode, gencode.(string), results, &wg, p.Version, sem)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	finalResults := make(map[string]string)
	for result := range results {
		for k, v := range result {
			finalResults[k] = getDomTree(v)
		}
	}

	for name, content := range finalResults {
		name = filepath.Join(saveDir, name)
		err = save(name, []byte(content))
		if err != nil {
			log.Printf("Error saving file: %v\n", err)
		}
		log.Printf("Saved file: %s\n", name)
	}

	return nil
}
