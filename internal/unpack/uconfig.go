package unpack

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Ackites/KillWxapkg/internal/enum"

	"github.com/Ackites/KillWxapkg/internal/config"

	"github.com/dop251/goja"
)

// ConfigParser 具体的配置文件解析器
type ConfigParser struct {
	OutputDir string
}

// PageConfig 存储页面配置
type PageConfig struct {
	Window map[string]interface{} `json:"window,omitempty"`
}

// AppConfig 存储应用配置
type AppConfig struct {
	Pages                          []string               `json:"pages"`
	Window                         map[string]interface{} `json:"window,omitempty"`
	TabBar                         map[string]interface{} `json:"tabBar,omitempty"`
	NetworkTimeout                 map[string]interface{} `json:"networkTimeout,omitempty"`
	SubPackages                    []SubPackage           `json:"subPackages,omitempty"`
	NavigateToMiniProgramAppIdList []string               `json:"navigateToMiniProgramAppIdList,omitempty"`
	Workers                        string                 `json:"workers,omitempty"`
	Debug                          bool                   `json:"debug,omitempty"`
}

// SubPackage 存储子包配置
type SubPackage struct {
	Root  string   `json:"root"`
	Pages []string `json:"pages"`
}

// changeExt 更改文件扩展名
func changeExt(filename, newExt string) string {
	ext := filepath.Ext(filename)
	return filename[:len(filename)-len(ext)] + newExt
}

// save 保存内容到文件
func save(filename string, content []byte) error {
	// 处理文件路径
	filename = filepath.ToSlash(filename)
	if idx := strings.Index(filename, ":"); idx != -1 {
		filename = filename[:idx+1] + strings.ReplaceAll(filename[idx+1:], ":", "")
	}

	// 判断目录是否存在
	dir := filepath.Dir(filename)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("unable to create directory %s: %v", dir, err)
		}
	}
	err := os.WriteFile(filename, content, 0755)
	if err != nil {
		return fmt.Errorf("unable to save file %s: %v", filename, err)
	}
	return nil
}

// Parse 解析和处理配置文件
func (p *ConfigParser) Parse(option config.WxapkgInfo) error {
	dir := filepath.Dir(option.Option.AppConfigSource)
	content, err := os.ReadFile(option.Option.AppConfigSource)
	if err != nil {
		return err
	}

	// 定义结构体以解析 JSON 内容
	var e struct {
		Pages                          []string               `json:"pages"`
		EntryPagePath                  string                 `json:"entryPagePath"`
		Global                         map[string]interface{} `json:"global"`
		TabBar                         map[string]interface{} `json:"tabBar"`
		NetworkTimeout                 map[string]interface{} `json:"networkTimeout"`
		SubPackages                    []SubPackage           `json:"subPackages"`
		NavigateToMiniProgramAppIdList []string               `json:"navigateToMiniProgramAppIdList"`
		ExtAppid                       string                 `json:"extAppid"`
		Ext                            map[string]interface{} `json:"ext"`
		Debug                          bool                   `json:"debug"`
		Page                           map[string]PageConfig  `json:"page"`
	}

	err = json.Unmarshal(content, &e)
	if err != nil {
		return err
	}

	// 处理页面路径，将 entryPagePath 放在首位
	k := e.Pages
	entryIndex := indexOf(k, changeExt(e.EntryPagePath, ""))
	k = append(k[:entryIndex], k[entryIndex+1:]...)
	k = append([]string{changeExt(e.EntryPagePath, "")}, k...)

	// 构建应用配置
	app := AppConfig{
		Pages:          k,
		Window:         e.Global["window"].(map[string]interface{}),
		TabBar:         e.TabBar,
		NetworkTimeout: e.NetworkTimeout,
	}

	// 处理子包
	if len(e.SubPackages) > 0 {
		var subPackages []SubPackage
		for _, subPackage := range e.SubPackages {
			root := subPackage.Root
			if !strings.HasSuffix(root, "/") {
				root += "/"
			}
			root = strings.TrimPrefix(root, "/")

			var newPages []string
			for i := 0; i < len(app.Pages); {
				page := app.Pages[i]
				if strings.HasPrefix(page, root) {
					newPage := strings.TrimPrefix(page, root)
					newPages = append(newPages, newPage)
					app.Pages = append(app.Pages[:i], app.Pages[i+1:]...)
				} else {
					i++
				}
			}

			// 去除重复的页面
			pageSet := make(map[string]struct{})
			var uniquePages []string
			for _, page := range newPages {
				if _, exists := pageSet[page]; !exists {
					pageSet[page] = struct{}{}
					uniquePages = append(uniquePages, page)
				}
			}
			newPages = uniquePages

			subPackage.Root = root
			if len(newPages) == 0 {
				subPackage.Pages = []string{}
			} else {
				subPackage.Pages = newPages
			}
			subPackages = append(subPackages, subPackage)
		}
		app.SubPackages = subPackages
		fmt.Printf("=======================================================\n这个小程序采用了分包\n子包个数为: %d\n=======================================================\n", len(app.SubPackages))
	}

	// 处理 navigateToMiniProgramAppIdList
	if len(e.NavigateToMiniProgramAppIdList) > 0 {
		app.NavigateToMiniProgramAppIdList = e.NavigateToMiniProgramAppIdList
	}

	// 处理 extAppid
	if len(e.ExtAppid) > 0 {
		extContent, _ := json.MarshalIndent(map[string]interface{}{
			"extEnable": true,
			"extAppid":  e.ExtAppid,
			"ext":       e.Ext,
		}, "", "    ")
		err := save(filepath.Join(dir, "ext.json"), extContent)
		if err != nil {
			return err
		}
	}

	// 处理调试模式
	if e.Debug {
		app.Debug = e.Debug
	}

	// 处理页面中的组件路径
	cur := "./file"
	for a, page := range e.Page {
		if page.Window != nil && page.Window["usingComponents"] != nil {
			for _, componentPath := range page.Window["usingComponents"].(map[string]interface{}) {
				componentPath := componentPath.(string) + ".html"
				file := componentPath
				if filepath.IsAbs(componentPath) {
					file = componentPath[1:]
				} else {
					file = toDir(filepath.Join(filepath.Dir(a), componentPath), cur)
				}
				if _, ok := e.Page[file]; !ok {
					e.Page[file] = PageConfig{}
				}
				if e.Page[file].Window == nil {
					e.Page[file] = PageConfig{Window: map[string]interface{}{"component": true}}
				}
			}
		}
	}

	// 处理 app-service.js 文件, 主包及子包
	if fileExists(filepath.Join(dir, enum.App_Service)) {
		serviceContent, _ := os.ReadFile(filepath.Join(dir, enum.App_Service))
		matches := findMatches(`__wxAppCode__\['[^']+\.json'\]\s*=\s*({[^;]*});`, string(serviceContent))
		if len(matches) > 0 {
			attachInfo := make(map[string]interface{})
			vm := goja.New()
			err = vm.Set("__wxAppCode__", attachInfo)
			if err != nil {
				return err
			}
			_, err = vm.RunString(strings.Join(matches, ""))
			if err != nil {
				return err
			}
			for name, info := range attachInfo {
				e.Page[changeExt(name, ".html")] = PageConfig{Window: info.(map[string]interface{})}
			}
		}

		// 子包配置 app-service.js
		for _, subPackage := range app.SubPackages {
			root := subPackage.Root
			subServiceFile := filepath.Join(dir, root, enum.App_Service)
			if !fileExists(subServiceFile) {
				continue
			}
			serviceContent, _ = os.ReadFile(subServiceFile)
			matches = findMatches(`__wxAppCode__\['[^']+\.json'\]\s*=\s*({[^;]*});`, string(serviceContent))
			if len(matches) > 0 {
				attachInfo := make(map[string]interface{})
				vm := goja.New()
				err := vm.Set("__wxAppCode__", attachInfo)
				if err != nil {
					return err
				}
				_, err = vm.RunString(strings.Join(matches, ""))
				if err != nil {
					return err
				}
				for name, info := range attachInfo {
					e.Page[changeExt(name, ".html")] = PageConfig{Window: info.(map[string]interface{})}
				}
			}
		}
	}

	// 保存页面 JSON 文件
	for a := range e.Page {
		aFile := changeExt(a, ".json")
		fileName := filepath.Join(dir, aFile)
		if aFile != "app.json" {
			windowContent, _ := json.MarshalIndent(e.Page[a].Window, "", "    ")
			err = save(fileName, windowContent)
			if err != nil {
				log.Printf("Error saving file %s: %v\n", fileName, err)
			}
		}
	}

	// 处理子包中的文件
	if len(app.SubPackages) > 0 {
		for _, subPackage := range app.SubPackages {
			for _, item := range subPackage.Pages {
				a := subPackage.Root + item + ".xx"
				err := save(filepath.Join(dir, changeExt(a, ".js")), []byte("// "+changeExt(a, ".js")+"\nPage({data: {}})"))
				if err != nil {
					return err
				}
				err = save(filepath.Join(dir, changeExt(a, ".wxml")), []byte("<!--"+changeExt(a, ".wxml")+"--><text>"+changeExt(a, ".wxml")+"</text>"))
				if err != nil {
					return err
				}
				err = save(filepath.Join(dir, changeExt(a, ".wxss")), []byte("/* "+changeExt(a, ".wxss")+" */"))
				if err != nil {
					return err
				}
			}
		}
	}

	// 处理 TabBar 图标路径
	if app.TabBar != nil && app.TabBar["list"] != nil {
		var digests [][2]interface{}
		for _, file := range scanDirByExt(dir, "") {
			data, _ := os.ReadFile(file)
			digests = append(digests, [2]interface{}{md5.Sum(data), file})
		}

		for _, e := range app.TabBar["list"].([]interface{}) {
			pagePath := e.(map[string]interface{})["pagePath"].(string)
			e.(map[string]interface{})["pagePath"] = changeExt(pagePath, "")
			if iconData, ok := e.(map[string]interface{})["iconData"].(string); ok {
				hash := md5.Sum([]byte(iconData))
				for _, digest := range digests {
					digestByte, _ := digest[0].([16]byte)
					if bytes.Equal(hash[:], digestByte[:]) {
						delete(e.(map[string]interface{}), "iconData")
						e.(map[string]interface{})["iconPath"] = fixDir(digest[1].(string), dir)
						break
					}
				}
			}
			if selectedIconData, ok := e.(map[string]interface{})["selectedIconData"].(string); ok {
				hash := md5.Sum([]byte(selectedIconData))
				for _, digest := range digests {
					digestByte, _ := digest[0].([16]byte)
					if bytes.Equal(hash[:], digestByte[:]) {
						delete(e.(map[string]interface{}), "selectedIconData")
						e.(map[string]interface{})["selectedIconPath"] = fixDir(digest[1].(string), dir)
						break
					}
				}
			}
		}
	}

	// 保存应用配置到 app.json
	appContent, _ := json.MarshalIndent(app, "", "    ")
	err = save(filepath.Join(dir, "app.json"), appContent)
	if err != nil {
		return err
	}
	log.Printf("Config file processed: %s\n", option.Option.AppConfigSource)
	return nil
}

// indexOf 返回字符串切片中项的索引
func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

// fileExists 检查文件是否存在
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// toDir 将文件路径转换为相对路径
func toDir(file, base string) string {
	relative, err := filepath.Rel(base, file)
	if err != nil {
		return file
	}
	return relative
}

// findMatches 查找所有匹配模式的字符串
func findMatches(pattern, text string) []string {
	re := regexp.MustCompile(pattern)
	return re.FindAllString(text, -1)
}

// scanDirByExt 扫描目录中的文件并返回指定扩展名的文件列表
func scanDirByExt(dir, ext string) []string {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(info.Name(), ext) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil
	}
	return files
}

// fixDir 修复文件路径为相对路径
func fixDir(file, base string) string {
	rel, err := filepath.Rel(base, file)
	if err != nil {
		return file
	}
	return filepath.ToSlash(rel)
}
