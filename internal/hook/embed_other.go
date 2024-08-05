//go:build !windows
// +build !windows

package hook

// 在非 Windows 平台下不嵌入任何内容
var embeddedExe []byte
