package unpack

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Ackites/KillWxapkg/internal/enum"

	"github.com/Ackites/KillWxapkg/internal/config"
	"github.com/dop251/goja"
)

// JavaScriptParser JavaScript 解析器
type JavaScriptParser struct {
	OutputDir string
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

	// 防止报错
	patch := `var window={};var navigator={};navigator.userAgent="iPhone";window.screen={};document={};function require(){};`

	scriptcode := patch + string(code)

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

	//// 提供 __g 变量的默认实现
	//err = vm.Set("__g", make(map[string]interface{}))
	//if err != nil {
	//	return err
	//}
	//
	//// 提供  __wxConfig 变量的默认实现
	//err = vm.Set("__wxConfig", make(map[string]interface{}))
	//if err != nil {
	//	return err
	//}
	//
	//// 提供 global 变量的默认实现
	//err = vm.Set("global", make(map[string]interface{}))
	//if err != nil {
	//	return err
	//}
	//
	//err = vm.Set("__vd_version_info__", map[string]interface{}{
	//	"version": "1.0.0",
	//	"build":   "default",
	//})
	//if err != nil {
	//	return err
	//}
	//
	//wxAppCode := make(map[string]func())
	//// 设置 __wxAppCode__
	//err = vm.Set("__wxAppCode__", wxAppCode)
	//if err != nil {
	//	log.Printf("Error setting __wxAppCode__: %v\n", err)
	//	return err
	//}

	// 设置 define 函数和 require 函数的行为
	err = vm.Set("define", func(call goja.FunctionCall) goja.Value {
		moduleName := call.Argument(0).String()
		funcBody := call.Argument(1).String()

		cleanedCode, err := removeWrapper(funcBody)
		if err != nil {
			log.Printf("Error removing wrapper: %v\n", err)
			cleanedCode = funcBody
		}

		//bcode := cleanedCode
		// 检查是否包含 "use strict" 并处理
		//if strings.HasPrefix(cleanedCode, `"use strict";`) || strings.HasPrefix(cleanedCode, `'use strict';`) {
		//	cleanedCode = cleanedCode[13:]
		//} else if (strings.HasPrefix(cleanedCode, `(function(){"use strict";`) || strings.HasPrefix(cleanedCode, `(function(){'use strict';`)) &&
		//	strings.HasSuffix(cleanedCode, `})();`) {
		//	cleanedCode = cleanedCode[25 : len(cleanedCode)-5]
		//}

		err = save(filepath.Join(dir, moduleName), []byte(cleanedCode))
		if err != nil {
			log.Printf("Error saving file: %v\n", err)
		}
		return goja.Undefined()
	})
	if err != nil {
		return err
	}

	//err = vm.Set("require", func(call goja.FunctionCall) goja.Value {
	//	// 返回一个空对象，表示对 require 的任何调用都将返回这个空对象
	//	result := vm.NewObject()
	//	return result
	//})
	//if err != nil {
	//	return err
	//}
	//
	//// 设置 $gwx 变量
	//err = vm.Set("$gwx", func(call goja.FunctionCall) goja.Value {
	//	// $gwx function implementation here
	//	return goja.Undefined()
	//})
	//if err != nil {
	//	return err
	//}

	_, err = vm.RunString(safeScript)
	if err != nil {
		return fmt.Errorf("failed to run JavaScript: %w", err)
	}

	log.Printf("Splitting \"%s\" done.", option.Option.ServiceSource)
	return nil
}
