package util

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

// 定义需要移除的供应商前缀
var removeTypes = []string{"webkit", "moz", "ms", "o"}

// TransformCSS 函数用于转换 CSS
func TransformCSS(style string) string {
	l := css.NewLexer(parse.NewInputString(style))
	var sb strings.Builder
	var inDeclarationBlock *bool = new(bool)

	for {
		tokenType, token := l.Next()
		if tokenType == css.ErrorToken {
			if l.Err() == io.EOF {
				break
			} else {
				fmt.Println("Error:", l.Err())
				return ""
			}
		}

		switch tokenType {
		case css.CommentToken:
			sb.WriteString(fmt.Sprintf("\n%s\n", token))
		case css.IdentToken:
			if *inDeclarationBlock {
				handleProperty(l, &sb, token, inDeclarationBlock)
			} else {
				handleSelector(&sb, token)
			}
		case css.LeftBraceToken:
			sb.WriteString(" {\n")
			*inDeclarationBlock = true
		case css.RightBraceToken:
			sb.WriteString("}\n")
			*inDeclarationBlock = false
		default:
			sb.WriteString(string(token))
		}
	}

	return sb.String()
}

// 处理选择器
func handleSelector(sb *strings.Builder, token []byte) {
	selector := strings.TrimSpace(string(token))
	if strings.HasPrefix(selector, "wx-") {
		selector = selector[3:]
	} else if selector == "body" {
		selector = "page"
	}
	sb.WriteString(selector)
}

// 处理属性名和值
func handleProperty(l *css.Lexer, sb *strings.Builder, token []byte, inDeclarationBlock *bool) {
	property := string(token)
	if skipProperty(property) {
		skipValue(l, sb, inDeclarationBlock)
		return
	}
	initialContent := sb.String() // 获取当前内容的拷贝
	sb.WriteString(fmt.Sprintf("    %s", property))
	readValue(l, sb, initialContent, inDeclarationBlock)
}

// 判断是否跳过属性
func skipProperty(prop string) bool {
	for _, prefix := range removeTypes {
		if strings.HasPrefix(prop, "-"+prefix+"-") {
			return true
		}
	}
	return false
}

// 跳过属性值
func skipValue(l *css.Lexer, sb *strings.Builder, inDeclarationBlock *bool) {
	for {
		tokenType, _ := l.Next()
		if tokenType == css.SemicolonToken {
			break
		}
		if tokenType == css.RightBraceToken {
			sb.WriteString("\n}\n")
			*inDeclarationBlock = false
			break
		}
	}
}

// 读取属性值
func readValue(l *css.Lexer, sb *strings.Builder, initialContent string, inDeclarationBlock *bool) {
	// 第一个冒号
	var colon int64 = 1
	var isLeftBrace bool = false
	var isRightBrace bool = false

	var value bytes.Buffer
	for {
		tokenType, token := l.Next()
		if tokenType == css.ColonToken && colon == 1 {
			colon++
			continue
		}
		if tokenType == css.SemicolonToken {
			break
		}
		if tokenType == css.LeftBraceToken {
			isLeftBrace = true
			break
		}
		if tokenType == css.RightBraceToken {
			isRightBrace = true
			break
		}
		value.Write(token)
	}
	if shouldRemoveValue(value.String()) {
		resetStringBuilder(sb, initialContent)
	} else {
		if isLeftBrace {
			sb.WriteString(" {\n")
			*inDeclarationBlock = true
			return
		} else {
			sb.WriteString(": " + value.String() + ";\n")
		}
		if isRightBrace {
			sb.WriteString("}\n")
			*inDeclarationBlock = false
		}
	}
}

// 重置 strings.Builder 的内容到指定长度
func resetStringBuilder(sb *strings.Builder, initialContent string) {
	sb.Reset()
	sb.WriteString(initialContent)
}

// 判断是否移除属性值
func shouldRemoveValue(value string) bool {
	if strings.HasPrefix(value, "progid:DXImageTransform") {
		return true
	}
	return false
}
