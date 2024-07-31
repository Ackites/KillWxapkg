package util

import (
	"os"
	"regexp"
)

// GetWccVersion 从源代码字符串中提取 __wcc_version__ 的值
func GetWccVersion(source string) string {
	if source == "" {
		return ""
	}

	// 读取source文件内容
	content, _ := os.ReadFile(source)

	// 定义正则表达式，用于匹配 __wcc_version__ 的值
	regex := regexp.MustCompile(`__wcc_version__\s*=\s*['"]([^'"]+)['"]`)

	// 查找匹配项
	matches := regex.FindStringSubmatch(string(content))

	// 如果匹配成功并捕获到版本号，则返回版本号
	if len(matches) > 1 {
		return matches[1]
	}

	// 未找到匹配项，返回空字符串
	return ""
}
