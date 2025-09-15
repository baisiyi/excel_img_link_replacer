package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// OpenFileDirectory 跨平台打开文件所在目录
// 在 Windows 上使用 explorer，在 macOS 上使用 open，在 Linux 上使用 xdg-open
func OpenFileDirectory(filePath string) error {
	dir := filepath.Dir(filePath)

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Windows: 使用 explorer 打开文件夹并选中文件
		cmd = exec.Command("explorer", "/select,", filePath)
	case "darwin":
		// macOS: 使用 open 命令打开文件夹
		cmd = exec.Command("open", "-R", filePath)
	case "linux":
		// Linux: 使用 xdg-open 打开文件夹
		cmd = exec.Command("xdg-open", dir)
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	return cmd.Run()
}

// OpenDirectory 仅打开目录（不选中特定文件）
func OpenDirectory(dirPath string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Windows: 使用 explorer 打开文件夹
		cmd = exec.Command("explorer", dirPath)
	case "darwin":
		// macOS: 使用 open 命令打开文件夹
		cmd = exec.Command("open", dirPath)
	case "linux":
		// Linux: 使用 xdg-open 打开文件夹
		cmd = exec.Command("xdg-open", dirPath)
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	return cmd.Run()
}

// FileExists 检查文件是否存在
func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}
