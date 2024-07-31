package formatter

import (
	"bytes"

	"github.com/ditashi/jsbeautifier-go/jsbeautifier"
)

// JSFormatter 结构体，用于格式化 JavaScript 代码
type JSFormatter struct{}

// NewJSFormatter 创建一个新的 JSFormatter 实例
func NewJSFormatter() *JSFormatter {
	return &JSFormatter{}
}

// Format 方法用于格式化 JavaScript 代码
// input: 原始的 JavaScript 代码字节切片
// 返回值: 格式化后的 JavaScript 代码字节切片和错误信息（如果有）
func (f *JSFormatter) Format(input []byte) ([]byte, error) {
	// 将输入数据转换为字符串，并去除前后空白
	code := string(bytes.TrimSpace(input))

	// 使用 jsbeautifier 库格式化 JavaScript 代码
	beautifiedCode, err := jsbeautifier.Beautify(&code, jsbeautifier.DefaultOptions())
	if err != nil {
		return input, err
	}

	return []byte(beautifiedCode), nil
}

func init() {
	RegisterFormatter(".js", NewJSFormatter())
}
