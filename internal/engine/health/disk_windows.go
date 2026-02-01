//go:build windows

package health

import (
	"fmt"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func FreeBytes(path string) (int64, error) {
	if path == "" {
		return 0, fmt.Errorf("disk free path missing")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return 0, fmt.Errorf("disk free abs: %w", err)
	}
	volume := filepath.VolumeName(abs)
	if volume == "" {
		return 0, fmt.Errorf("disk free volume missing")
	}
	root := volume + "\\"
	var freeBytes uint64
	if err := windows.GetDiskFreeSpaceEx(windows.StringToUTF16Ptr(root), &freeBytes, nil, nil); err != nil {
		return 0, fmt.Errorf("disk free: %w", err)
	}
	return int64(freeBytes), nil
}
