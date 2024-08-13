package unpack

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dop251/goja"

	"github.com/Ackites/KillWxapkg/internal/enum"

	"github.com/Ackites/KillWxapkg/internal/config"
)

// JavaScriptParser JavaScript 解析器
type JavaScriptParser struct {
	OutputDir string
}

// DefineParams 存储从 define 函数中提取的参数
type DefineParams struct {
	ModuleName string
	FuncBody   string
}

func cleanDefineFunc(jsCode string) string {
	// 正则表达式匹配 define 函数的头部
	reHead := regexp.MustCompile(`^define\s*\(\s*["'].*?["']\s*,\s*function\s*\([^)]*\)\s*\{`)

	// 正则表达式匹配 define 函数的尾部
	reTail := regexp.MustCompile(`\}\s*,\s*\{[^}]*isPage\s*:\s*[^}]*\}\s*\)\s*;$`)

	// 移除头部
	cleanedCode := reHead.ReplaceAllString(jsCode, "")

	// 移除尾部
	cleanedCode = reTail.ReplaceAllString(cleanedCode, "")

	// 去除开头和结尾的空白字符
	cleanedCode = strings.TrimSpace(cleanedCode)

	// 去除"严格模式"声明
	if strings.HasPrefix(cleanedCode, `"use strict";`) || strings.HasPrefix(cleanedCode, `'use strict';`) {
		cleanedCode = cleanedCode[13:]
		cleanedCode = strings.TrimSpace(cleanedCode)
	}

	return cleanedCode
}

// extractDefineParams 提取所有 define 函数的第一个和第二个参数
func extractDefineParams(jsCode string) ([]DefineParams, error) {
	// 正则表达式提取 define 函数的第一个和第二个参数
	re := regexp.MustCompile(`define\s*\(\s*["']([^"']+)["']\s*,\s*function\s*\(([^)]*)\)\s*\{([\s\S]*?)\}\s*,\s*\{[^}]*isPage\s*:\s*[^}]*\}\s*\)\s*;`)
	matches := re.FindAllStringSubmatch(jsCode, -1)

	if len(matches) == 0 {
		results, err := run(jsCode)
		if err != nil {
			return nil, err
		}
		return results, nil
	}

	var results = make([]DefineParams, 0)
	for _, match := range matches {
		if len(match) >= 3 {
			params := DefineParams{
				ModuleName: match[1],
				FuncBody:   cleanDefineFunc(match[0]),
			}
			results = append(results, params)
		}
	}
	return results, nil
}

func run(code string) ([]DefineParams, error) {
	var results = make([]DefineParams, 0)
	// 防止报错
	patch := `var window={};var navigator={};navigator.userAgent="iPhone";window.screen={};
document={getElementsByTagName:()=>{}};function require(){};var global={};var __wxAppCode__={};var __wxConfig={};
var __vd_version_info__={};var $gwx=function(){};var __g={};`

	scriptcode := patch + code

	// 包裹 try...catch 语句以捕获 JavaScript 错误
	safeScript := `
	try {
		` + string(scriptcode) + `
	} catch (e) {
		//console.error(e);
	}
	`

	vm := goja.New()

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

	// 设置 define 函数和 require 函数的行为
	err := vm.Set("define", func(call goja.FunctionCall) goja.Value {
		moduleName := call.Argument(0).String()
		funcBody := call.Argument(1).String()

		cleanedCode, err := removeWrapper(funcBody)
		if err != nil {
			log.Printf("Error removing wrapper: %v\n", err)
			cleanedCode = funcBody
		}

		//检查是否包含 "use strict" 并处理
		if strings.HasPrefix(cleanedCode, `"use strict";`) || strings.HasPrefix(cleanedCode, `'use strict';`) {
			cleanedCode = cleanedCode[13:]
		} else if (strings.HasPrefix(cleanedCode, `(function(){"use strict";`) || strings.HasPrefix(cleanedCode, `(function(){'use strict';`)) &&
			strings.HasSuffix(cleanedCode, `})();`) {
			cleanedCode = cleanedCode[25 : len(cleanedCode)-5]
		}

		params := DefineParams{
			ModuleName: moduleName,
			FuncBody:   cleanedCode,
		}
		results = append(results, params)

		return goja.Undefined()
	})
	if err != nil {
		return nil, err
	}

	_, err = vm.RunString(safeScript)
	if err != nil {
		return nil, fmt.Errorf("failed to run JavaScript: %w", err)
	}

	return results, nil
}

// removeWrapper 移除函数包装器
func removeWrapper(jsCode string) (string, error) {
	vm := goja.New()
	script := `
		(function(code) {
			let match = code.match(/^function\s*\(.*?\)\s*\{([\s\S]*)\}$/);
			if (match && match[1]) {
				// 每一行缩进减少一个空格
				match[1] = match[1].trim();
				code = match[1].replace(/^\s{4}/gm, '');
			}
			return code;
		})(code);
	`
	// 设置 JavaScript 变量
	err := vm.Set("code", jsCode)
	if err != nil {
		return "", err
	}
	value, err := vm.RunString(script)
	if err != nil {
		return "", fmt.Errorf("JavaScript execution error: %w", err)
	}
	return value.String(), nil
}

// 是否为分包
func isSubpackage(wxapkg *config.WxapkgInfo) bool {
	switch wxapkg.WxapkgType {
	case enum.APP_SUBPACKAGE_V1, enum.APP_SUBPACKAGE_V2, enum.GAME_SUBPACKAGE:
		return true
	default:
		return false
	}
}

// Parse 解析和分割 JavaScript 文件
func (p *JavaScriptParser) Parse(option config.WxapkgInfo) error {

	dir := option.SourcePath
	if isSubpackage(&option) {
		dir = p.OutputDir
	}

	code, err := os.ReadFile(option.Option.ServiceSource)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	params, err := extractDefineParams(string(code))
	if err != nil {
		return err
	}

	for _, param := range params {
		err = save(filepath.Join(dir, param.ModuleName), []byte(param.FuncBody))
		if err != nil {
			log.Printf("Error saving file: %v\n", err)
		}
	}

	log.Printf("Splitting \"%s\" done.", option.Option.ServiceSource)
	return nil
}
