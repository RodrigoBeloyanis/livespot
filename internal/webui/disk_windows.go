//go:build windows

package webui

import "golang.org/x/sys/windows"

func getDiskFreeSpaceEx(path string, freeBytes *int64) error {
	var free uint64
	if err := windows.GetDiskFreeSpaceEx(windows.StringToUTF16Ptr(path), &free, nil, nil); err != nil {
		return err
	}
	*freeBytes = int64(free)
	return nil
}
