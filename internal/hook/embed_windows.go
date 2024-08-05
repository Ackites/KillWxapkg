//go:build windows
// +build windows

package hook

import (
	_ "embed"
)

//go:embed win.exe
var embeddedExe []byte
