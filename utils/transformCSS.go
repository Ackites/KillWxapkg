package utils

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/gorilla/css/scanner"
)

// 定义需要移除的供应商前缀
var removeTypes = []string{"webkit", "moz", "ms", "o"}

// Declaration 结构体用于存储 CSS 声明
type Declaration struct {
	Property string
	Value    string
}

// TransformCSS 函数用于转换 CSS
func TransformCSS(style string) string {
	s := scanner.New(style)
	var sb strings.Builder
	var currentSelector string
	var currentDeclarations []Declaration

	for {
		token := s.Next()
		if token.Type == scanner.TokenEOF {
			if currentSelector != "" {
				writeRuleset(&sb, currentSelector, currentDeclarations)
			}
			break
		}

		switch token.Type {
		case scanner.TokenS:
			// 忽略空白
			continue
		case scanner.TokenComment:
			sb.WriteString(fmt.Sprintf("\n%s\n", token.Value))
		case scanner.TokenIdent:
			// 处理选择器或属性名
			if currentSelector == "" {
				// 选择器
				currentSelector = strings.TrimSpace(token.Value)
				if strings.HasPrefix(currentSelector, "wx-") {
					currentSelector = currentSelector[3:]
				} else if currentSelector == "body" {
					currentSelector = "page"
				}
			} else {
				// 属性名
				currentDeclarations = append(currentDeclarations, readDeclaration(s, token.Value))
			}
		case scanner.TokenChar:
			if token.Value == "{" {
				// 忽略大括号
				continue
			} else if token.Value == "}" {
				// 遇到右大括号，写入规则集
				writeRuleset(&sb, currentSelector, currentDeclarations)
				currentSelector = ""
				currentDeclarations = nil
			}
		default:
			panic("unhandled default case")
		}
	}

	return beautifyCSS(sb.String())
}

// readDeclaration 函数读取一个声明
func readDeclaration(s *scanner.Scanner, property string) Declaration {
	var value bytes.Buffer
	foundColon := false
	// 1: 遇到冒号，跳过冒号
	// 2: 遇到冒号，不跳过冒号
	count := 1

	for {
		token := s.Next()
		if token.Type == scanner.TokenEOF || token.Value == "}" || token.Value == ";" {
			break
		}

		if token.Value == ":" && count == 1 {
			foundColon = true
			count++
			continue
		}

		if foundColon {
			value.WriteString(token.Value)
		}
	}

	prop := strings.TrimSpace(property)
	val := strings.TrimSpace(value.String())

	if shouldRemoveProperty(prop, val) {
		return Declaration{}
	}

	return Declaration{
		Property: prop,
		Value:    val,
	}
}

// shouldRemoveProperty 函数判断是否应移除属性
func shouldRemoveProperty(prop, value string) bool {
	// 移除包含 progid:DXImageTransform 的值
	if strings.HasPrefix(value, "progid:DXImageTransform") {
		return true
	}
	// 移除指定前缀的属性
	for _, prefix := range removeTypes {
		if strings.HasPrefix(prop, "-"+prefix+"-") {
			return true
		}
		if strings.HasPrefix(value, "-"+prefix+"-") {
			return true
		}
	}
	return false
}

// writeRuleset 函数用于写入处理后的规则集
func writeRuleset(sb *strings.Builder, selector string, declarations []Declaration) {
	sb.WriteString(selector + " {\n")
	for _, decl := range declarations {
		// 过滤空的声明
		if decl.Property != "" && decl.Value != "" {
			sb.WriteString(fmt.Sprintf("    %s: %s;\n", decl.Property, decl.Value))
		}
	}
	sb.WriteString("}\n\n")
}

// beautifyCSS 函数用于美化 CSS
func beautifyCSS(css string) string {
	var beautified strings.Builder
	indent := 0
	lines := strings.Split(css, "\n")

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		if strings.HasPrefix(trimmedLine, "}") {
			indent--
		}

		beautified.WriteString(strings.Repeat("    ", indent))
		beautified.WriteString(trimmedLine)
		beautified.WriteString("\n")

		if strings.HasSuffix(trimmedLine, "{") {
			indent++
		}
	}

	return beautified.String()
}
