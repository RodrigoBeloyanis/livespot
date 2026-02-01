//go:build !windows

package health

import "fmt"

func FreeBytes(path string) (int64, error) {
	return 0, fmt.Errorf("disk free unsupported on this platform")
}
