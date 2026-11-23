package core

import "golang.org/x/sys/windows"

var (
	user32           = windows.NewLazySystemDLL("user32.dll")
	getSystemMetrics = user32.NewProc("GetSystemMetrics")
)

const (
	smCXScreen = 0
	smCYScreen = 1
)

// PrimaryScreenSize replaces System.Windows.Forms.Screen.PrimaryScreen.Bounds.
func PrimaryScreenSize() (int, int) {
	w, _, _ := getSystemMetrics.Call(smCXScreen)
	h, _, _ := getSystemMetrics.Call(smCYScreen)
	return int(w), int(h)
}
