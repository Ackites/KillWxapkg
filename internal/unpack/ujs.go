package unpack

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	. "github.com/Ackites/KillWxapkg/internal/config"

	"github.com/dop251/goja"
)

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

// removeInvalidLineCode 删除无效行代码
func removeInvalidLineCode(code string) string {
	invalidRe := regexp.MustCompile(`\s+[a-z] = VM2_INTERNAL_STATE_DO_NOT_USE_OR_PROGRAM_WILL_FAIL\.handleException\([a-z]\);`)
	return invalidRe.ReplaceAllString(code, "")
}

// SplitJs 解析和分割 JavaScript 文件
func SplitJs(filePath string, mainDir string, wg *sync.WaitGroup, errChan chan error) {
	defer wg.Done()

	isSubPkg := mainDir != ""
	dir := filepath.Dir(filePath)
	if isSubPkg {
		dir = mainDir
	}

	code, err := os.ReadFile(filePath)
	if err != nil {
		errChan <- fmt.Errorf("failed to read file: %w", err)
		return
	}

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
	_ = vm.Set("console", console)

	// 提供 __g 变量的默认实现
	err = vm.Set("__g", make(map[string]interface{}))
	if err != nil {
		errChan <- err
		return
	}

	err = vm.Set("__vd_version_info__", map[string]interface{}{
		"version": "1.0.0",
		"build":   "default",
	})
	if err != nil {
		errChan <- err
		return
	}

	wxAppCode := make(map[string]func())
	// 设置 __wxAppCode__
	err = vm.Set("__wxAppCode__", wxAppCode)
	if err != nil {
		log.Printf("Error setting __wxAppCode__: %v\n", err)
		return
	}

	// 设置 define 函数和 require 函数的行为
	err = vm.Set("define", func(call goja.FunctionCall) goja.Value {
		moduleName := call.Argument(0).String()
		funcBody := call.Argument(1).String()

		cleanedCode, err := removeWrapper(funcBody)
		if err != nil {
			log.Printf("Error removing wrapper: %v\n", err)
			cleanedCode = funcBody
		}

		bcode := cleanedCode
		// 检查是否包含 "use strict" 并处理
		if strings.HasPrefix(cleanedCode, `"use strict";`) || strings.HasPrefix(cleanedCode, `'use strict';`) {
			cleanedCode = cleanedCode[13:]
		} else if (strings.HasPrefix(cleanedCode, `(function(){"use strict";`) || strings.HasPrefix(cleanedCode, `(function(){'use strict';`)) &&
			strings.HasSuffix(cleanedCode, `})();`) {
			cleanedCode = cleanedCode[25 : len(cleanedCode)-5]
		}

		// 删除无效行代码
		res := removeInvalidLineCode(cleanedCode)
		if res == "" {
			log.Printf("Fail to delete 'use strict' in \"%s\".", moduleName)
			res = removeInvalidLineCode(bcode)
		}

		err = saveToFile(filepath.Join(dir, moduleName), []byte(res))
		if err != nil {
			log.Printf("Error saving file: %v\n", err)
		}
		return goja.Undefined()
	})
	if err != nil {
		errChan <- err
		return
	}

	err = vm.Set("require", func(call goja.FunctionCall) goja.Value {
		// 返回一个空对象，表示对 require 的任何调用都将返回这个空对象
		result := vm.NewObject()
		return result
	})
	if err != nil {
		errChan <- err
		return
	}

	if isSubPkg {
		codeStr := string(code)
		code = []byte(codeStr[strings.Index(codeStr, "define("):])
	}

	_, err = vm.RunString(string(code))
	if err != nil {
		errChan <- fmt.Errorf("failed to run JavaScript: %w", err)
		return
	}
	manager := NewFileDeletionManager()
	manager.AddFile(filePath)
	log.Printf("Splitting \"%s\" done.", filePath)
}

// saveToFile 保存文件内容
func saveToFile(filePath string, content []byte) error {
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}
	err = os.WriteFile(filePath, content, 0755)
	if err != nil {
		log.Printf("Save file error: %v\n", err)
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

// ProcessJavaScriptFiles 处理所有 JavaScript 文件
func ProcessJavaScriptFiles(dir string, config struct {
	SubPackages []SubPackage `json:"subPackages"`
}) error {
	var wg sync.WaitGroup
	errChan := make(chan error, 10) // 缓冲区大小可以根据需要调整

	// 处理主包
	appServicePath := filepath.Join(dir, "app-service.js")
	workersPath := filepath.Join(dir, "workers.js")
	if _, err := os.Stat(appServicePath); err == nil {
		wg.Add(1)
		go SplitJs(appServicePath, "", &wg, errChan)
	}
	if _, err := os.Stat(workersPath); err == nil {
		wg.Add(1)
		go SplitJs(workersPath, "", &wg, errChan)
	}

	// 遍历所有子包
	for _, subPackage := range config.SubPackages {
		subDir := filepath.Join(dir, subPackage.Root)
		if _, err := os.Stat(subDir); err != nil {
			continue
		}
		appServicePath = filepath.Join(subDir, "app-service.js")
		workersPath = filepath.Join(subDir, "workers.js")
		if _, err := os.Stat(appServicePath); err == nil {
			wg.Add(1)
			go SplitJs(appServicePath, dir, &wg, errChan)
		}
		if _, err := os.Stat(workersPath); err == nil {
			wg.Add(1)
			go SplitJs(workersPath, dir, &wg, errChan)
		}
	}

	// 等待所有goroutine完成
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// 处理错误
	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}
