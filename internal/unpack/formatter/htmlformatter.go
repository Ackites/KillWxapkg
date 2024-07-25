package formatter

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/yosssi/gohtml"
)

// HTMLFormatter 结构体，用于格式化 HTML 代码
type HTMLFormatter struct {
	jsFormatter *JSFormatter
}

// 正则表达式用于匹配 HTML 中的 <script> 标签及其内容
var regScriptInHtml = regexp.MustCompile(`(?s) *<script.*?>(.*?)</script>`)

// NewHTMLFormatter 创建一个新的 HTMLFormatter 实例
func NewHTMLFormatter() *HTMLFormatter {
	return &HTMLFormatter{
		jsFormatter: NewJSFormatter(),
	}
}

// Format 方法用于格式化 HTML 代码
// input: 原始的 HTML 代码字节切片
// 返回值: 格式化后的 HTML 代码字节切片和错误信息（如果有）
func (f *HTMLFormatter) Format(input []byte) ([]byte, error) {
	// 使用 gohtml 库格式化 HTML 代码
	data := gohtml.FormatBytes(bytes.TrimSpace(input))

	// 替换 <script> 标签中的 JavaScript 代码
	data = regScriptInHtml.ReplaceAllFunc(data, func(script []byte) []byte {
		// 计算 <script> 标签前的空格数量
		space := countLeadingSpaces(script)

		// 提取 <script> 标签中的 JavaScript 代码
		jsCode := regScriptInHtml.FindSubmatch(script)[1]
		jsStr := strings.Repeat(" ", space+2) + string(bytes.TrimSpace(jsCode))

		// 使用 jsFormatter 格式化 JavaScript 代码
		beautifiedCode, err := f.jsFormatter.Format([]byte(jsStr))
		if err == nil {
			// 替换原始 JavaScript 代码为格式化后的代码，保持缩进一致性
			return bytes.Replace(script, jsCode, []byte("\n"+string(beautifiedCode)+"\n"+strings.Repeat(" ", space)), 1)
		}
		return script
	})

	return data, nil
}

// countLeadingSpaces 函数用于计算字节切片前导空格的数量
func countLeadingSpaces(data []byte) int {
	result := 0
	for _, c := range data {
		if c == ' ' {
			result++
		} else {
			break
		}
	}
	return result
}

func init() {
	RegisterFormatter(".html", NewHTMLFormatter())
}
