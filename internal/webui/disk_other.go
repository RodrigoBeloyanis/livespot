//go:build !windows

package webui

func getDiskFreeSpaceEx(path string, freeBytes *int64) error {
	*freeBytes = 0
	return nil
}
