//go:build !linux && !darwin

package output

func ioctlWidth(fd uintptr) int {
	return 0
}
